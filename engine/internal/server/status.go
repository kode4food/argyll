package server

import (
	"maps"

	"github.com/kode4food/timebox/raft"
)

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

func NewRaftStatusProvider(p *raft.Persistence) StatusProvider {
	return func() map[string]any {
		if p == nil {
			return nil
		}
		addr, id := p.LeaderWithID()
		return map[string]any{
			"backend": map[string]any{
				"type":           "raft",
				"state":          p.State(),
				"leader_address": addr,
				"leader_id":      id,
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
