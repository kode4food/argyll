package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	ErrGetCatalogState  = errors.New("failed to get catalog state")
	ErrGetClusterState  = errors.New("failed to get cluster state")
	ErrGetCatalogEvents = errors.New("failed to get catalog events")
	ErrGetClusterEvents = errors.New("failed to get cluster events")
)

func (s *Server) handleEngine(c *gin.Context) {
	cat, err := s.engine.GetCatalogState()
	if writeError(c, ErrGetCatalogState, err) {
		return
	}
	cluster, err := s.engine.GetClusterState()
	if writeError(c, ErrGetClusterState, err) {
		return
	}

	last := cat.LastUpdated
	if cluster.LastUpdated.After(last) {
		last = cluster.LastUpdated
	}
	c.JSON(http.StatusOK, gin.H{
		"last_updated": last,
		"steps":        cat.Steps,
		"attributes":   cat.Attributes,
		"health":       completeClusterHealth(cat, cluster).Nodes,
	})
}

func (s *Server) getCatalog(c *gin.Context) {
	cat, err := s.engine.GetCatalogState()
	writeValue(c, ErrGetCatalogState, cat, err)
}

func (s *Server) getCluster(c *gin.Context) {
	cluster, err := s.engine.GetClusterState()
	writeValue(c, ErrGetClusterState, cluster, err)
}

func (s *Server) getCatalogEvents(c *gin.Context) {
	evs, err := s.engine.GetCatalogEvents()
	writeEvents(c, ErrGetCatalogEvents, evs, err)
}

func (s *Server) getClusterEvents(c *gin.Context) {
	evs, err := s.engine.GetClusterEvents()
	writeEvents(c, ErrGetClusterEvents, evs, err)
}

func completeClusterHealth(
	cat api.CatalogState, cluster api.ClusterState,
) api.ClusterState {
	res := cluster
	for nid, node := range cluster.Nodes {
		nextNode := node
		changed := false
		for sid := range cat.Steps {
			if _, ok := nextNode.Health[sid]; ok {
				continue
			}
			nextNode = nextNode.SetHealth(sid, api.HealthState{
				Status: api.HealthUnknown,
			})
			changed = true
		}
		if changed {
			res = res.SetNode(nid, nextNode)
		}
	}
	return res
}

func writeValue[T any](c *gin.Context, sentinel error, val T, err error) {
	if !writeError(c, sentinel, err) {
		c.JSON(http.StatusOK, val)
	}
}

func writeEvents[E any](c *gin.Context, sentinel error, evs []E, err error) {
	if !writeError(c, sentinel, err) {
		c.JSON(http.StatusOK, gin.H{"events": evs, "count": len(evs)})
	}
}

func writeError(c *gin.Context, sentinel, err error) bool {
	if err == nil {
		return false
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", sentinel, err),
		Status: http.StatusInternalServerError,
	})
	return true
}
