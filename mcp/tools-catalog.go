package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	spaceInput struct {
		Space map[string]any `json:"space" jsonschema:"Space definition"`
	}

	updateSpaceInput struct {
		Space map[string]any `json:"space" jsonschema:"Whole Space definition"`
		ID    api.SpaceID    `json:"id" jsonschema:"Space ID"`
	}

	feedInput struct {
		Feed map[string]any `json:"feed" jsonschema:"Feed definition"`
	}

	updateFeedInput struct {
		Feed map[string]any `json:"feed" jsonschema:"Whole Feed definition"`
		ID   api.FeedID     `json:"id" jsonschema:"Feed ID"`
	}
)

const (
	spaceShape = "A Space has id, name, optional description, and a selector " +
		`{"language": "argyll", "source": "..."} that receives the Step as value, such as ` +
		`"\"domain:risk\" in value.tags". The Step has tags, type, handling, and attributes ` +
		"keyed by name"
	feedShape = `A Feed has id, name, optional description, value (the seed as a JSON literal, ` +
		`such as "0"), attributes (the output Attribute names it folds), and fold ` +
		`{"language": "argyll", "source": "..."}. The fold runs once per contributing Step with ` +
		"feed bound to the current value and each matching output bound by name, and returns " +
		"the next value"
)

func (s *Server) registerSpaceTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name:        "list_spaces",
		Description: "List Spaces. A Space is a named selection of Steps a Flow can plan with",
	},
		pathTool(s.httpGet, "/engine/spaces"),
	)
	addTool(reg, &mcp.Tool{
		Name:        "get_space",
		Description: "Fetch one Space with its selector",
	}, idTool(s.httpGet, "/engine/spaces/%s"))
	addTool(reg, &mcp.Tool{
		Name:        "list_space_steps",
		Description: "List the Steps a Space selects",
	}, idTool(s.httpGet, "/engine/spaces/%s/steps"))
	addTool(reg, &mcp.Tool{
		Name:        "unregister_space",
		Description: "Remove a Space",
	}, idTool(s.httpDelete, "/engine/spaces/%s"))
	addTool(reg, &mcp.Tool{
		Name:        "preview_space",
		Description: "Show which Steps a Space would select without saving it. " + spaceShape,
	},
		func(ctx context.Context, args spaceInput) (any, error) {
			if len(args.Space) == 0 {
				return nil, errInvalidParams("space body is required")
			}
			return s.httpPost(ctx, "/engine/spaces/preview", args.Space)
		},
	)
	addTool(reg, &mcp.Tool{
		Name:        "register_space",
		Description: "Register a new Space. " + spaceShape,
	},
		func(ctx context.Context, args spaceInput) (any, error) {
			if len(args.Space) == 0 {
				return nil, errInvalidParams("space body is required")
			}
			return s.httpPost(ctx, "/engine/spaces", args.Space)
		},
	)
	addTool(reg, &mcp.Tool{
		Name: "update_space",
		Description: "Replace an existing Space. Fetch it with get_space, change it, and send " +
			"the whole Space",
	},
		func(ctx context.Context, args updateSpaceInput) (any, error) {
			if args.ID == "" || len(args.Space) == 0 {
				return nil, errInvalidParams("id and space body are required")
			}
			return s.httpPut(ctx, "/engine/spaces/"+string(args.ID), args.Space)
		},
	)
}

func (s *Server) registerFeedTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name:        "list_feeds",
		Description: "List Feeds. A Feed is a catalog-wide value folded from settled Step outputs",
	},
		pathTool(s.httpGet, "/engine/feeds"),
	)
	addTool(reg, &mcp.Tool{
		Name:        "get_feed",
		Description: "Fetch one Feed with its current value",
	}, idTool(s.httpGet, "/engine/feeds/%s"))
	addTool(reg, &mcp.Tool{
		Name:        "unregister_feed",
		Description: "Remove a Feed",
	}, idTool(s.httpDelete, "/engine/feeds/%s"))
	addTool(reg, &mcp.Tool{
		Name:        "register_feed",
		Description: "Register a new Feed. " + feedShape,
	},
		func(ctx context.Context, args feedInput) (any, error) {
			if len(args.Feed) == 0 {
				return nil, errInvalidParams("feed body is required")
			}
			return s.httpPost(ctx, "/engine/feeds", args.Feed)
		},
	)
	addTool(reg, &mcp.Tool{
		Name: "update_feed",
		Description: "Replace an existing Feed. Fetch it with get_feed, change it, and send the " +
			"whole Feed",
	},
		func(ctx context.Context, args updateFeedInput) (any, error) {
			if args.ID == "" || len(args.Feed) == 0 {
				return nil, errInvalidParams("id and feed body are required")
			}
			return s.httpPut(ctx, "/engine/feeds/"+string(args.ID), args.Feed)
		},
	)
}
