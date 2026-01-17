package mcp

import (
	"encoding/json"

	"github.com/deinstapel/go-jsonrpc"
)

const rpcErrTool = -32000

func toolResult(payload any, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, jsonrpc.NewRPCErr(rpcErrTool, err.Error())
	}
	return marshalResult(toolCallResult{
		Content: []toolContent{
			{Type: "json", JSON: payload},
		},
	})
}

func marshalResult(result any) (json.RawMessage, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}
