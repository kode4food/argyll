package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	ErrListSpaces      = errors.New("failed to list spaces")
	ErrPreviewSpace    = errors.New("failed to preview space")
	ErrRegisterSpace   = errors.New("failed to register space")
	ErrUnregisterSpace = errors.New("failed to unregister space")
)

func (s *Server) listSpaces(c *gin.Context) {
	spaces, err := s.engine.ListSpaces()
	writeValue(c, ErrListSpaces, api.SpacesListResponse{
		Spaces: spaces,
		Count:  len(spaces),
	}, err)
}

func (s *Server) createSpace(c *gin.Context) {
	sp, ok := bindSpace(c)
	if !ok {
		return
	}
	sp.ID = api.SanitizeID(sp.ID)
	sp = sp.Normalize()
	err := s.engine.RegisterSpace(sp)
	if err == nil {
		c.JSON(http.StatusCreated, api.SpaceRegisteredResponse{
			Space:   sp,
			Message: "Space registered",
		})
		return
	}
	if errors.Is(err, engine.ErrSpaceExists) {
		c.JSON(http.StatusConflict, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusConflict,
		})
		return
	}
	if errors.Is(err, engine.ErrInvalidSpace) {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrRegisterSpace, err),
		Status: http.StatusInternalServerError,
	})
}

func (s *Server) previewSpace(c *gin.Context) {
	sp, ok := bindSpace(c)
	if !ok {
		return
	}
	preview, err := s.engine.PreviewSpace(sp)
	if errors.Is(err, engine.ErrInvalidSpace) {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		})
		return
	}
	writeValue(c, ErrPreviewSpace, preview, err)
}

func (s *Server) getSpace(c *gin.Context) {
	cat, err := s.engine.GetCatalogState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetCatalogState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}
	id := api.SpaceID(c.Param("space_id"))
	if sp, ok := cat.Spaces[id]; ok {
		c.JSON(http.StatusOK, sp)
		return
	}
	c.JSON(http.StatusNotFound, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %s", engine.ErrSpaceNotFound, id),
		Status: http.StatusNotFound,
	})
}

func (s *Server) listSpaceSteps(c *gin.Context) {
	cat, err := s.engine.GetCatalogState()
	if writeError(c, ErrGetCatalogState, err) {
		return
	}
	id := api.SpaceID(c.Param("space_id"))
	if _, ok := cat.Spaces[id]; !ok {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", engine.ErrSpaceNotFound, id),
			Status: http.StatusNotFound,
		})
		return
	}
	selected := cat.SpaceSteps(id)
	steps := make([]*api.Step, 0, len(selected))
	for _, st := range selected {
		steps = append(steps, st)
	}
	c.JSON(http.StatusOK, api.StepsListResponse{
		Steps: steps,
		Count: len(steps),
	})
}

func (s *Server) updateSpace(c *gin.Context) {
	sp, ok := bindSpace(c)
	if !ok {
		return
	}
	sp.ID = api.SanitizeID(sp.ID)
	sp = sp.Normalize()
	id := api.SanitizeID(api.SpaceID(c.Param("space_id")))
	if sp.ID != id {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Space ID in URL does not match space ID in body",
			Status: http.StatusBadRequest,
		})
		return
	}
	err := s.engine.UpdateSpace(sp)
	if err == nil {
		c.JSON(http.StatusOK, api.SpaceRegisteredResponse{
			Space:   sp,
			Message: "Space updated",
		})
		return
	}
	if errors.Is(err, engine.ErrSpaceNotFound) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusNotFound,
		})
		return
	}
	if errors.Is(err, engine.ErrSpaceGoalExcluded) {
		c.JSON(http.StatusConflict, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusConflict,
		})
		return
	}
	if errors.Is(err, engine.ErrInvalidSpace) {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("failed to update space: %v", err),
		Status: http.StatusInternalServerError,
	})
}

func (s *Server) deleteSpace(c *gin.Context) {
	id := api.SpaceID(c.Param("space_id"))
	err := s.engine.UnregisterSpace(id)
	if err == nil {
		c.JSON(http.StatusOK, api.MessageResponse{
			Message: "Space unregistered",
		})
		return
	}
	if errors.Is(err, engine.ErrSpaceNotFound) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusNotFound,
		})
		return
	}
	if errors.Is(err, engine.ErrSpaceInUse) {
		c.JSON(http.StatusConflict, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusConflict,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrUnregisterSpace, err),
		Status: http.StatusInternalServerError,
	})
}

func bindSpace(c *gin.Context) (api.Space, bool) {
	c.Request.Body = http.MaxBytesReader(
		c.Writer, c.Request.Body, MaxStepBodyBytes,
	)
	var sp api.Space
	if err := c.ShouldBindJSON(&sp); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return api.Space{}, false
	}
	return sp, true
}
