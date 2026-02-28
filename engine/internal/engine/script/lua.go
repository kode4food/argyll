package script

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/Shopify/go-lua"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// LuaEnv provides a Lua script execution environment with state pooling
	LuaEnv struct {
		*compiler[*CompiledLua]
		statePool chan *lua.State
	}

	// CompiledLua represents a compiled Lua script
	CompiledLua struct {
		bytecode []byte
		argNames []string
	}
)

const (
	luaCacheSize        = 4096
	luaStatePoolSize    = 10
	luaGlobalTableIndex = -2
	luaArrayTableIndex  = -3
	luaMapTableIndex    = -3
	luaArgLocalTemplate = "local %s = select(%d, ...)"
	luaGlobalTableName  = "_G"
	luaSeparator        = "\n"
)

var (
	ErrLuaLoad      = errors.New("lua load error")
	ErrLuaExecution = errors.New("lua execution error")
)

var luaExclude = [...]string{
	"io", "os", "debug", "package", "require", "dofile", "loadfile", "load",
}

// NewLuaEnv creates a new Lua script execution environment with a state pool
// for efficient script reuse
func NewLuaEnv() *LuaEnv {
	luaEnv := &LuaEnv{
		statePool: make(chan *lua.State, luaStatePoolSize),
	}
	luaEnv.compiler = newCompiler(luaCacheSize,
		func(step *api.Step, cfg *api.ScriptConfig) (*CompiledLua, error) {
			argNames := step.SortedArgNames()
			src := luaEnv.wrapSource(cfg.Script, argNames)
			return luaEnv.compile(src, argNames)
		},
	)
	return luaEnv
}

// ExecuteScript runs a compiled Lua script with the provided inputs and
// returns the output arguments
func (e *LuaEnv) ExecuteScript(
	c Compiled, _ *api.Step, inputs api.Args,
) (api.Args, error) {
	proc := c.(*CompiledLua)
	var result api.Args
	err := e.withCompiledResult(proc, inputs,
		func(L *lua.State) {
			if L.IsTable(-1) {
				result = luaTableToMap(L, -1)
			} else {
				value := luaToGo(L, -1)
				result = api.Args{"result": value}
			}
			L.Pop(1)
		},
	)
	return result, err
}

// EvaluatePredicate executes a compiled Lua predicate with the provided inputs
// and returns the boolean result
func (e *LuaEnv) EvaluatePredicate(
	c Compiled, _ *api.Step, inputs api.Args,
) (bool, error) {
	proc := c.(*CompiledLua)
	result := false
	err := e.withCompiledResult(proc, inputs,
		func(L *lua.State) {
			result = L.ToBoolean(-1)
			L.Pop(1)
		},
	)
	return result, err
}

func (e *LuaEnv) wrapSource(script string, argNames []string) string {
	argLocals := make([]string, len(argNames))
	for i, name := range argNames {
		argLocals[i] = fmt.Sprintf(luaArgLocalTemplate, name, i+1)
	}
	return strings.Join([]string{
		strings.Join(argLocals, luaSeparator), script,
	}, luaSeparator)
}

func (e *LuaEnv) compile(src string, argNames []string) (*CompiledLua, error) {
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

func (e *LuaEnv) withCompiledResult(
	proc *CompiledLua, inputs api.Args, onResult func(*lua.State),
) error {
	L := e.getState()
	defer e.returnState(L)

	e.setupSandbox(L)
	if err := L.Load(bytes.NewReader(proc.bytecode), "chunk", "b"); err != nil {
		return fmt.Errorf("%w: %w", ErrLuaLoad, err)
	}

	for _, name := range proc.argNames {
		pushLuaArg(L, inputs, name)
	}

	if err := L.ProtectedCall(len(proc.argNames), 1, 0); err != nil {
		return fmt.Errorf("%w: %w", ErrLuaExecution, err)
	}

	onResult(L)
	return nil
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
