package server

import (
	"encoding/json"
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
	if writeError(c, writeErrorArgs{
		sentinel: ErrGetCatalogState,
		err:      err,
	}) {
		return
	}
	cluster, err := s.engine.GetClusterState()
	if writeError(c, writeErrorArgs{
		sentinel: ErrGetClusterState,
		err:      err,
	}) {
		return
	}

	last := cat.LastUpdated
	if cluster.LastUpdated.After(last) {
		last = cluster.LastUpdated
	}
	c.JSON(http.StatusOK, gin.H{
		"last_updated": last,
		"steps":        cat.Steps,
		"spaces":       cat.Spaces,
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

func completeClusterHealth(cat api.CatalogState, cluster api.ClusterState) api.ClusterState {
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
	if !writeError(c, writeErrorArgs{
		sentinel: sentinel,
		err:      err,
	}) {
		c.JSON(http.StatusOK, val)
	}
}

func writeEvents[E any](c *gin.Context, sentinel error, evs []E, err error) {
	if !writeError(c, writeErrorArgs{
		sentinel: sentinel,
		err:      err,
	}) {
		c.JSON(http.StatusOK, gin.H{
			"events": evs,
			"count":  len(evs),
		})
	}
}

// bindJSON binds a request body of at most limit bytes and returns a sanitized copy
func bindJSON[T api.Sanitizer[T]](c *gin.Context, limit int64) (T, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	var v T
	if err := json.NewDecoder(c.Request.Body).Decode(&v); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return v, false
	}
	return v.Sanitized(), true
}

// sanitizedParam reads a path parameter as a sanitized ID
func sanitizedParam[T ~string](c *gin.Context, name string) T {
	return api.SanitizeID(T(c.Param(name)))
}

type writeErrorArgs struct {
	sentinel error
	err      error
}

func writeError(c *gin.Context, args writeErrorArgs) bool {
	if args.err == nil {
		return false
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", args.sentinel, args.err),
		Status: http.StatusInternalServerError,
	})
	return true
}
