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
	ErrListFeeds      = errors.New("failed to list feeds")
	ErrGetFeed        = errors.New("failed to get feed")
	ErrRegisterFeed   = errors.New("failed to register feed")
	ErrUpdateFeed     = errors.New("failed to update feed")
	ErrUnregisterFeed = errors.New("failed to unregister feed")
)

func (s *Server) listFeeds(c *gin.Context) {
	feeds, err := s.engine.ListFeeds()
	writeValue(c, ErrListFeeds, api.FeedsListResponse{
		Feeds: feeds,
		Count: len(feeds),
	}, err)
}

func (s *Server) createFeed(c *gin.Context) {
	f, ok := bindFeed(c)
	if !ok {
		return
	}
	f.ID = api.SanitizeID(f.ID)
	err := s.engine.RegisterFeed(f)
	if err == nil {
		c.JSON(http.StatusCreated, api.FeedRegisteredResponse{
			Feed:    f,
			Message: "Feed registered",
		})
		return
	}
	if errors.Is(err, api.ErrFeedExists) {
		writeFeedStatus(c, http.StatusConflict, err)
		return
	}
	if errors.Is(err, api.ErrInvalidFeed) {
		writeFeedStatus(c, http.StatusBadRequest, err)
		return
	}
	writeError(c, writeErrorArgs{sentinel: ErrRegisterFeed, err: err})
}

func (s *Server) getFeed(c *gin.Context) {
	res, err := s.engine.GetFeed(api.FeedID(c.Param("feed_id")))
	if err == nil {
		c.JSON(http.StatusOK, res)
		return
	}
	if errors.Is(err, api.ErrFeedNotFound) {
		writeFeedStatus(c, http.StatusNotFound, err)
		return
	}
	writeError(c, writeErrorArgs{sentinel: ErrGetFeed, err: err})
}

func (s *Server) updateFeed(c *gin.Context) {
	f, ok := bindFeed(c)
	if !ok {
		return
	}
	f.ID = api.SanitizeID(f.ID)
	id := api.SanitizeID(api.FeedID(c.Param("feed_id")))
	if f.ID != id {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Feed ID in URL does not match Feed ID in body",
			Status: http.StatusBadRequest,
		})
		return
	}
	err := s.engine.UpdateFeed(f)
	if err == nil {
		c.JSON(http.StatusOK, api.FeedRegisteredResponse{
			Feed:    f,
			Message: "Feed updated",
		})
		return
	}
	if errors.Is(err, api.ErrFeedNotFound) {
		writeFeedStatus(c, http.StatusNotFound, err)
		return
	}
	if errors.Is(err, api.ErrInvalidFeed) {
		writeFeedStatus(c, http.StatusBadRequest, err)
		return
	}
	writeError(c, writeErrorArgs{sentinel: ErrUpdateFeed, err: err})
}

func (s *Server) deleteFeed(c *gin.Context) {
	err := s.engine.UnregisterFeed(api.FeedID(c.Param("feed_id")))
	if err == nil {
		c.JSON(http.StatusOK, api.MessageResponse{
			Message: "Feed unregistered",
		})
		return
	}
	if errors.Is(err, api.ErrFeedNotFound) {
		writeFeedStatus(c, http.StatusNotFound, err)
		return
	}
	if errors.Is(err, engine.ErrFeedInUse) {
		writeFeedStatus(c, http.StatusConflict, err)
		return
	}
	writeError(c, writeErrorArgs{sentinel: ErrUnregisterFeed, err: err})
}

func writeFeedStatus(c *gin.Context, status int, err error) {
	c.JSON(status, api.ErrorResponse{Error: err.Error(), Status: status})
}

func bindFeed(c *gin.Context) (*api.Feed, bool) {
	c.Request.Body = http.MaxBytesReader(
		c.Writer, c.Request.Body, MaxStepBodyBytes,
	)
	f := &api.Feed{}
	if err := c.ShouldBindJSON(f); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return nil, false
	}
	return f, true
}
