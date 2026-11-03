package server_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func testFeed() api.Feed {
	return api.Feed{
		ID:    "order-total",
		Name:  "Order Total",
		Value: "0",
		Fold: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return feed + amount",
		},
		Attributes: api.Slice[api.Name]{"amount"},
	}
}

func TestFeedAPI(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		router := env.Server.SetupRoutes()
		do := func(method, path string, body any) int {
			w := spaceRequest(t, spaceRequestArgs{
				handler: router,
				method:  method,
				path:    path,
				body:    body,
			})
			return w.Code
		}
		f := testFeed()
		charge := helpers.NewStepWithOutputs("charge", "amount")
		assert.NoError(t, env.Engine.RegisterStep(charge))

		assert.Equal(t,
			http.StatusCreated, do(http.MethodPost, "/engine/feeds", f))
		clash := api.Feed{
			ID:         f.ID,
			Name:       f.Name,
			Value:      "1",
			Fold:       f.Fold,
			Attributes: f.Attributes,
		}
		assert.Equal(t,
			http.StatusConflict, do(http.MethodPost, "/engine/feeds", clash))
		bad := api.Feed{ID: "bad"}
		assert.Equal(t,
			http.StatusBadRequest, do(http.MethodPost, "/engine/feeds", bad))

		w := spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodGet,
			path:    "/engine/feeds/order-total",
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var got api.FeedResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Equal(t, f.ID, got.Feed.ID)
		assert.EqualValues(t, 0, got.State.Value)
		assert.NotEmpty(t, got.State.Version)

		w = spaceRequest(t, spaceRequestArgs{
			handler: router,
			method:  http.MethodGet,
			path:    "/engine/feeds",
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var listed api.FeedsListResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &listed))
		assert.Equal(t, 1, listed.Count)

		assert.Equal(t,
			http.StatusNotFound,
			do(http.MethodGet, "/engine/feeds/missing", nil))

		f.Name = "Renamed"
		assert.Equal(t,
			http.StatusOK, do(http.MethodPut, "/engine/feeds/order-total", f))
		assert.Equal(t,
			http.StatusBadRequest, do(http.MethodPut, "/engine/feeds/other", f))
		other := testFeed()
		other.ID = "other"
		assert.Equal(t,
			http.StatusNotFound,
			do(http.MethodPut, "/engine/feeds/other", other))
		f.Fold = &api.ScriptConfig{Language: api.ScriptLangLua}
		assert.Equal(t,
			http.StatusBadRequest,
			do(http.MethodPut, "/engine/feeds/order-total", f))

		reader := helpers.NewSimpleStep("reader")
		reader.Attributes["total"] = &api.AttributeSpec{
			Role: api.RoleFeed,
			Feed: &api.FeedConfig{ID: "order-total"},
		}
		assert.NoError(t, env.Engine.RegisterStep(reader))
		assert.Equal(t,
			http.StatusConflict,
			do(http.MethodDelete, "/engine/feeds/order-total", nil))
		assert.NoError(t, env.Engine.UnregisterStep(reader.ID))

		assert.Equal(t,
			http.StatusOK,
			do(http.MethodDelete, "/engine/feeds/order-total", nil))
		assert.Equal(t,
			http.StatusNotFound,
			do(http.MethodDelete, "/engine/feeds/order-total", nil))
		assert.Equal(t,
			http.StatusBadRequest,
			do(http.MethodPost, "/engine/feeds", "not a feed"))
	})
}
