package script

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/Shopify/go-lua"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// LuaEnv provides a Lua script execution environment with state pooling
	LuaEnv struct {
		*compiler[*compiledLua]
		statePool  chan *luaState
		prelude    []byte
		preludeErr error
	}

	// compiledLua represents compiled Lua source
	compiledLua struct {
		bytecode []byte
		names    luaNames
	}

	// luaNames holds the derived aliases added to this, and the locals bound to clean names or
	// derived tails
	luaNames struct {
		aliases []bareName
		locals  []bareName
	}

	// luaState is a pooled state and the number of chunks it has loaded
	luaState struct {
		state  *lua.State
		chunks int
	}
)

const (
	luaCacheSize        = 4096
	luaStatePoolSize    = 10
	luaChunkPoolSize    = 256
	luaGlobalTableIndex = -2
	luaArrayTableIndex  = -3
	luaMapTableIndex    = -3
	luaChunkTableIndex  = -4
	luaArgLocalTemplate = "local %s = select(%d, ...)"
	luaReceiver         = "this"
	luaGlobalTableName  = "_G"
	luaChunkTableName   = "argyll_chunks"
	luaSeparator        = "\n"
	luaEnvUpValue       = 1
)

// helper functions setupSandbox installs as globals for every evaluation
//
//go:embed prelude.lua
var luaPreludeSource string

var (
	ErrLuaLoad      = errors.New("lua load error")
	ErrLuaExecution = errors.New("lua execution error")
)

// _G reaches the globals table directly and getmetatable reaches a library table behind its
// stand-in, so both would outlive the call
var luaExclude = [...]string{
	"io", "os", "debug", "package", "require", "dofile", "loadfile", "load",
	luaGlobalTableName, "getmetatable",
}

// NewLuaEnv creates a new Lua script execution environment with a state pool for efficient reuse
func NewLuaEnv() *LuaEnv {
	prelude, err := compileLuaPrelude()
	luaEnv := &LuaEnv{
		statePool:  make(chan *luaState, luaStatePoolSize),
		prelude:    prelude,
		preludeErr: err,
	}
	luaEnv.compiler = newCompiler(luaCacheSize,
		func(st *api.Step, cfg *api.ScriptConfig) (*compiledLua, error) {
			return luaEnv.compile(cfg.Source, st.SortedArgNames())
		},
	)
	return luaEnv
}

// Evaluate runs compiled Lua source with the provided inputs and returns its result
func (e *LuaEnv) Evaluate(c Compiled, _ *api.Step, inputs api.Args) (any, error) {
	var result any
	err := e.withCompiledResult(c, inputs,
		func(L *lua.State) {
			result = luaToGo(L, -1)
			L.Pop(1)
		},
	)
	return result, err
}

func (*LuaEnv) wrapSource(src string, argNames []bareName) string {
	argLocals := make([]string, len(argNames)+1)
	argLocals[0] = fmt.Sprintf(luaArgLocalTemplate, luaReceiver, 1)
	for i, name := range argNames {
		argLocals[i+1] = fmt.Sprintf(luaArgLocalTemplate, name.name, i+2)
	}
	return strings.Join([]string{strings.Join(argLocals, luaSeparator), src}, luaSeparator)
}

func (e *LuaEnv) compile(src string, argNames []string) (*compiledLua, error) {
	L := lua.NewState()

	e.setupSandbox(L)

	names := newLuaNames(L, argNames)
	code, err := dumpChunk(L, e.wrapSource(src, names.locals))
	if err != nil {
		return nil, err
	}

	return &compiledLua{
		bytecode: code,
		names:    names,
	}, nil
}

// runs once per state, since the prelude's stand-ins keep evaluated source from reaching anything
// it installs
func (*LuaEnv) setupSandbox(L *lua.State) {
	lua.OpenLibraries(L)
	L.Global(luaGlobalTableName)
	for _, name := range luaExclude {
		L.PushNil()
		L.SetField(luaGlobalTableIndex, name)
	}
	L.Pop(1)
}

// evaluated source's private env cannot write to the globals, so the helpers are installed once per
// state rather than repaired on every call
func (e *LuaEnv) installPrelude(L *lua.State) error {
	if e.preludeErr != nil {
		return e.preludeErr
	}
	if err := L.Load(bytes.NewReader(e.prelude), "prelude", "b"); err != nil {
		return errors.Join(ErrLuaLoad, err)
	}
	if err := L.ProtectedCall(0, 0, 0); err != nil {
		return errors.Join(ErrLuaExecution, err)
	}
	return nil
}

func (e *LuaEnv) withCompiledResult(
	c Compiled, inputs api.Args, onResult func(*lua.State),
) error {
	proc := c.(*compiledLua)
	st, err := e.getState()
	if err != nil {
		return err
	}
	defer e.returnState(st)

	L := st.state
	if err := st.pushChunk(proc); err != nil {
		return err
	}
	if err := setChunkEnv(L); err != nil {
		return err
	}

	// locals read the values already converted onto the receiver
	pushLuaReceiver(L, withAliases(inputs, proc.names.aliases))
	recv := L.Top()
	for _, b := range proc.names.locals {
		L.PushString(b.input)
		L.RawGet(recv)
	}

	if err := L.ProtectedCall(len(proc.names.locals)+1, 1, 0); err != nil {
		return errors.Join(ErrLuaExecution, err)
	}

	onResult(L)
	return nil
}

func (e *LuaEnv) getState() (*luaState, error) {
	select {
	case st := <-e.statePool:
		return st, nil
	default:
	}
	L := lua.NewState()
	e.setupSandbox(L)
	if err := e.installPrelude(L); err != nil {
		return nil, err
	}
	st := &luaState{state: L}
	st.resetChunks()
	return st, nil
}

func (e *LuaEnv) returnState(st *luaState) {
	st.state.SetTop(0)

	select {
	case e.statePool <- st:
	default:
	}
}

// a chunk stays loaded on the state that ran it, so later calls reuse the closure instead of
// reading the bytecode again
func (st *luaState) pushChunk(proc *compiledLua) error {
	L := st.state
	L.Field(lua.RegistryIndex, luaChunkTableName)
	L.RawGetValue(-1, proc)
	if L.IsFunction(-1) {
		L.Remove(-2)
		return nil
	}
	L.Pop(1)

	if st.chunks >= luaChunkPoolSize {
		L.Pop(1)
		st.resetChunks()
		L.Field(lua.RegistryIndex, luaChunkTableName)
	}
	if err := L.Load(bytes.NewReader(proc.bytecode), "chunk", "b"); err != nil {
		L.Pop(1)
		return errors.Join(ErrLuaLoad, err)
	}
	L.PushLightUserData(proc)
	L.PushValue(-2)
	L.RawSet(luaChunkTableIndex)
	L.Remove(-2)
	st.chunks++
	return nil
}

// a cached chunk pins its source, so a state holding enough of them drops the whole table rather
// than tracking each one's last use
func (st *luaState) resetChunks() {
	st.state.NewTable()
	st.state.SetField(lua.RegistryIndex, luaChunkTableName)
	st.chunks = 0
}

// compiled once, so each state loads bytecode rather than parsing the prelude on every call
func compileLuaPrelude() ([]byte, error) {
	L := lua.NewState()
	lua.OpenLibraries(L)
	code, err := dumpChunk(L, luaPreludeSource)
	if err != nil {
		return nil, errors.Join(ErrLuaLoad, err)
	}
	return code, nil
}

func dumpChunk(L *lua.State, src string) ([]byte, error) {
	if err := lua.LoadString(L, src); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := L.Dump(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// gives the chunk on the stack a private _ENV that reads through to the globals, so its writes are
// discarded with the call
func setChunkEnv(L *lua.State) error {
	L.NewTable()
	// Raw writes bypass __newindex, so library stand-ins belong to the call
	for _, name := range []string{"string", "table", "math", "bit32"} {
		L.Global(name)
		if !L.MetaTable(-1) {
			L.Pop(1)
			continue
		}
		L.Remove(-2)
		L.NewTable()
		L.Insert(-2)
		L.SetMetaTable(-2)
		L.SetField(-2, name)
	}
	L.NewTable()
	L.PushGlobalTable()
	L.SetField(-2, "__index")
	L.PushBoolean(false)
	L.SetField(-2, "__metatable")
	L.SetMetaTable(-2)
	// a main chunk always carries _ENV, so this only fires if that changes
	if _, ok := lua.SetUpValue(L, -2, luaEnvUpValue); !ok {
		L.Pop(1)
		return ErrLuaLoad
	}
	return nil
}

func pushLuaReceiver(L *lua.State, args api.Args) {
	L.CreateTable(0, len(args))
	for name, value := range args {
		goToLua(L, value)
		L.SetField(-2, string(name))
	}
}

func newLuaNames(L *lua.State, names []string) luaNames {
	locals := bareNames(names, func(name string) bool {
		return isLuaName(L, name)
	})
	return luaNames{aliases: aliasesOf(locals), locals: locals}
}

// identPattern keeps a name from carrying any other source, so the parser alone decides which names
// are reserved words
func isLuaName(L *lua.State, name string) bool {
	if name == luaReceiver || !identPattern.MatchString(name) {
		return false
	}
	if err := lua.LoadString(L, "local "+name); err != nil {
		return false
	}
	L.Pop(1)
	return true
}

func goToLua(L *lua.State, value any) {
	switch v := value.(type) {
	case string:
		L.PushString(v)
	case bool:
		L.PushBoolean(v)
	case int:
		L.PushInteger(v)
	case int64:
		L.PushInteger(int(v))
	case float64:
		L.PushNumber(v)
	case []any:
		pushLuaArray(L, v)
	case map[string]any:
		pushLuaMap(L, v)
	case nil:
		L.PushNil()
	default:
		L.PushString(fmt.Sprintf("%v", v))
	}
}

func pushLuaArray(L *lua.State, arr []any) {
	L.CreateTable(len(arr), 0)
	for i, item := range arr {
		L.PushInteger(i + 1)
		goToLua(L, item)
		L.SetTable(luaArrayTableIndex)
	}
}

func pushLuaMap(L *lua.State, m map[string]any) {
	L.CreateTable(0, len(m))
	for k, val := range m {
		L.PushString(k)
		goToLua(L, val)
		L.SetTable(luaMapTableIndex)
	}
}

func luaNumberToGo(L *lua.State, index int) any {
	num, _ := L.ToNumber(index)
	if num == float64(int(num)) {
		return int(num)
	}
	return num
}

func luaToGo(L *lua.State, index int) any {
	switch L.TypeOf(index) {
	case lua.TypeNil:
		return nil
	case lua.TypeBoolean:
		return L.ToBoolean(index)
	case lua.TypeNumber:
		return luaNumberToGo(L, index)
	case lua.TypeString:
		s, _ := L.ToString(index)
		return s
	case lua.TypeTable:
		return luaTableToAny(L, index)
	default:
		return nil
	}
}

func luaTableToAny(L *lua.State, index int) any {
	absIndex := luaAbsIndex(L, index)
	isArray := true
	length := 0

	L.PushNil()
	for L.Next(absIndex) {
		if !L.IsNumber(-2) {
			isArray = false
			L.Pop(2)
			break
		}
		length++
		L.Pop(1)
	}

	if isArray && length > 0 {
		return convertLuaArray(L, convertLuaArrayArgs{
			index:  absIndex,
			length: length,
		})
	}

	result := map[string]any{}
	L.PushNil()
	for L.Next(absIndex) {
		var key string
		if !L.IsString(-2) {
			key = fmt.Sprintf("%v", luaToGo(L, -2))
			result[key] = luaToGo(L, -1)
			L.Pop(1)
			continue
		}
		key, _ = L.ToString(-2)
		result[key] = luaToGo(L, -1)
		L.Pop(1)
	}

	return result
}

type convertLuaArrayArgs struct {
	index  int
	length int
}

func convertLuaArray(L *lua.State, args convertLuaArrayArgs) []any {
	arr := make([]any, args.length)
	absIndex := luaAbsIndex(L, args.index)
	for i := 1; i <= args.length; i++ {
		L.RawGetInt(absIndex, i)
		arr[i-1] = luaToGo(L, -1)
		L.Pop(1)
	}
	return arr
}

func luaAbsIndex(L *lua.State, index int) int {
	if index >= 0 {
		return index
	}
	return L.Top() + index + 1
}
