package engine

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Shopify/go-lua"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	// LuaEnv provides a Lua script execution environment with state pooling
	LuaEnv struct {
		statePool chan *lua.State
		scripts   sync.Map
	}

	// CompiledLua represents a compiled Lua script
	CompiledLua struct {
		bytecode []byte
		argNames []string
	}
)

const (
	luaStatePoolSize    = 10
	luaGlobalTableIndex = -2
	luaArrayTableIndex  = -3
	luaMapTableIndex    = -3
	luaArgLocalTemplate = "local %s = select(%d, ...)"
	luaScriptSeparator  = "\n"
	luaGlobalTableName  = "_G"
)

var (
	ErrLuaBadCompiledType = errors.New("expected *CompiledLua")
	ErrLuaLoad            = errors.New("lua load error")
	ErrLuaExecution       = errors.New("lua execution error")
)

var luaExclude = [...]string{
	"io", "os", "debug", "package", "require", "dofile", "loadfile", "load",
}

// NewLuaEnv creates a new Lua script execution environment with a state pool
// for efficient script reuse
func NewLuaEnv() *LuaEnv {
	return &LuaEnv{
		statePool: make(chan *lua.State, luaStatePoolSize),
	}
}

// Compile compiles a Lua script with the given argument names, returning the
// compiled form or an error
func (e *LuaEnv) Compile(
	step *api.Step, script string, argNames []string,
) (Compiled, error) {
	if script == "" {
		return nil, nil
	}

	key := scriptCacheKey(step.ID, script)

	if val, ok := e.scripts.Load(key); ok {
		return val.(*CompiledLua), nil
	}

	c, err := e.compile(script, argNames)
	if err == nil {
		e.scripts.Store(key, c)
	}
	return c, err
}

// CompileStepScript compiles the main script for a step, extracting and
// ordering argument names automatically
func (e *LuaEnv) CompileStepScript(step *api.Step) (Compiled, error) {
	names := step.SortedArgNames()
	return e.compileScript(step.ID, scriptType, step.Script.Script, names)
}

// CompileStepPredicate compiles the predicate script for a step, which
// determines if the step should execute
func (e *LuaEnv) CompileStepPredicate(step *api.Step) (Compiled, error) {
	names := step.SortedArgNames()
	return e.compileScript(
		step.ID, predicateType, step.Predicate.Script, names,
	)
}

func (e *LuaEnv) compileScript(
	stepID api.StepID, scriptType, script string, argNames []string,
) (*CompiledLua, error) {
	key := scriptCacheKey(stepID, script)

	if val, ok := e.scripts.Load(key); ok {
		return val.(*CompiledLua), nil
	}

	c, err := e.compile(script, argNames)
	if err != nil {
		return nil, fmt.Errorf("step %s %s: %w", stepID, scriptType, err)
	}

	e.scripts.Store(key, c)
	return c, nil
}

// Validate checks if a Lua script is syntactically correct without running it
func (e *LuaEnv) Validate(step *api.Step, script string) error {
	names := step.SortedArgNames()
	_, err := e.compile(script, names)
	return err
}

// ExecuteScript runs a compiled Lua script with the provided inputs and
// returns the output arguments
func (e *LuaEnv) ExecuteScript(
	c Compiled, _ *api.Step, inputs api.Args,
) (api.Args, error) {
	script, ok := c.(*CompiledLua)
	if !ok {
		return nil, fmt.Errorf("%w, got %T", ErrLuaBadCompiledType, c)
	}

	result, err := executeLuaScript(e, script, inputs)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// EvaluatePredicate executes a compiled Lua predicate with the provided inputs
// and returns the boolean result
func (e *LuaEnv) EvaluatePredicate(
	c Compiled, _ *api.Step, inputs api.Args,
) (bool, error) {
	script, ok := c.(*CompiledLua)
	if !ok {
		return false, fmt.Errorf("%s, got %T", ErrLuaBadCompiledType, c)
	}

	return evaluateLuaPredicate(e, script, inputs)
}

func (e *LuaEnv) compile(
	script string, argNames []string,
) (*CompiledLua, error) {
	argLocals := make([]string, len(argNames))
	for i, name := range argNames {
		argLocals[i] = fmt.Sprintf(luaArgLocalTemplate, name, i+1)
	}

	src := strings.Join([]string{
		strings.Join(argLocals, luaScriptSeparator), script,
	}, luaScriptSeparator)

	L := lua.NewState()

	e.setupSandbox(L)

	if err := lua.LoadString(L, src); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := L.Dump(&buf); err != nil {
		return nil, err
	}

	return &CompiledLua{
		bytecode: buf.Bytes(),
		argNames: argNames,
	}, nil
}

func (e *LuaEnv) setupSandbox(L *lua.State) {
	lua.OpenLibraries(L)
	L.Global(luaGlobalTableName)
	for _, name := range luaExclude {
		L.PushNil()
		L.SetField(luaGlobalTableIndex, name)
	}
	L.Pop(1)
}

func (e *LuaEnv) getState() *lua.State {
	select {
	case L := <-e.statePool:
		return L
	default:
		return lua.NewState()
	}
}

func (e *LuaEnv) returnState(L *lua.State) {
	L.SetTop(0)

	select {
	case e.statePool <- L:
	default:
	}
}

func executeLuaScript(
	env *LuaEnv, c *CompiledLua, inputs api.Args,
) (api.Args, error) {
	L := env.getState()
	defer env.returnState(L)

	env.setupSandbox(L)
	if err := L.Load(bytes.NewReader(c.bytecode), "chunk", "b"); err != nil {
		return nil, err
	}

	for _, name := range c.argNames {
		pushLuaArg(L, inputs, name)
	}

	if err := L.ProtectedCall(len(c.argNames), 1, 0); err != nil {
		return nil, err
	}

	var result api.Args
	if L.IsTable(-1) {
		result = luaTableToMap(L, -1)
	} else {
		value := luaToGo(L, -1)
		result = api.Args{"result": value}
	}
	L.Pop(1)

	return result, nil
}

func evaluateLuaPredicate(
	env *LuaEnv, c *CompiledLua, inputs api.Args,
) (bool, error) {
	L := env.getState()
	defer env.returnState(L)

	env.setupSandbox(L)

	if err := L.Load(bytes.NewReader(c.bytecode), "chunk", "b"); err != nil {
		return false, fmt.Errorf("%w: %w", ErrLuaLoad, err)
	}

	for _, name := range c.argNames {
		pushLuaArg(L, inputs, name)
	}

	if err := L.ProtectedCall(len(c.argNames), 1, 0); err != nil {
		return false, fmt.Errorf("%w: %w", ErrLuaExecution, err)
	}

	result := L.ToBoolean(-1)
	L.Pop(1)

	return result, nil
}

func pushLuaArg(L *lua.State, inputs api.Args, argName string) {
	if value, ok := inputs[api.Name(argName)]; ok {
		goToLua(L, value)
		return
	}
	L.PushNil()
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
	switch {
	case L.IsNil(index):
		return nil
	case L.IsBoolean(index):
		return L.ToBoolean(index)
	case L.IsNumber(index):
		return luaNumberToGo(L, index)
	case L.IsString(index):
		s, _ := L.ToString(index)
		return s
	case L.IsTable(index):
		return luaTableToAny(L, index)
	default:
		return nil
	}
}

func luaTableToMap(L *lua.State, index int) api.Args {
	result := api.Args{}

	L.PushNil()
	for L.Next(index - 1) {
		if L.IsString(-2) {
			key, _ := L.ToString(-2)
			result[api.Name(key)] = luaToGo(L, -1)
		}
		L.Pop(1)
	}

	return result
}

func luaTableToAny(L *lua.State, index int) any {
	isArray := true
	length := 0

	L.PushNil()
	for L.Next(index - 1) {
		if !L.IsNumber(-2) {
			isArray = false
			L.Pop(1)
			break
		}
		length++
		L.Pop(1)
	}

	if isArray && length > 0 {
		return convertLuaArray(L, index, length)
	}

	result := map[string]any{}
	L.PushNil()
	for L.Next(index - 1) {
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

func convertLuaArray(L *lua.State, index, length int) []any {
	arr := make([]any, length)
	absIndex := index
	if index < 0 {
		absIndex = L.Top() + index + 1
	}
	for i := 1; i <= length; i++ {
		L.RawGetInt(absIndex, i)
		arr[i-1] = luaToGo(L, -1)
		L.Pop(1)
	}
	return arr
}
