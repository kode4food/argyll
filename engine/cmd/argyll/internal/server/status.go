package server

import "maps"

type StatusProvider func() map[string]any

func NewWebSocketStatusProvider(s *Server) StatusProvider {
	return func() map[string]any {
		return map[string]any{
			"websocket": map[string]any{
				"clients": s.webSocketCount(),
			},
		}
	}
}

func (s *Server) statusDetails() map[string]any {
	res := map[string]any{}
	for _, getStatus := range s.status {
		if st := getStatus(); st != nil {
			maps.Copy(res, st)
		}
	}
	return res
}
