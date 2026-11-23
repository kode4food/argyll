package mcp

import "github.com/localrivet/gomcp/server"

type (
	spaceInput struct {
		Space map[string]any `json:"space"`
	}

	updateSpaceInput struct {
		Space map[string]any `json:"space"`
		ID    string         `json:"id"`
	}

	feedInput struct {
		Feed map[string]any `json:"feed"`
	}

	updateFeedInput struct {
		Feed map[string]any `json:"feed"`
		ID   string         `json:"id"`
	}
)

const (
	spaceShape = "A Space has id, name, optional description, and a selector " +
		`{"language": "argyll", "source": "..."} that receives the Step as value, such as ` +
		`"\"domain:risk\" in value.tags". The Step has tags, type, handling, and ` +
		"attributes keyed by name"
	feedShape = "A Feed has id, name, optional description, value (the seed as a JSON " +
		`literal, such as "0"), attributes (the output Attribute names it folds), and fold ` +
		`{"language": "argyll", "source": "..."}. The fold runs once per contributing Step ` +
		"with feed bound to the current value and each matching output bound by name, and " +
		"returns the next value"
)

func (s *Server) registerSpaceTools(srv server.Server) {
	srv.Tool(
		"list_spaces",
		"List Spaces. A Space is a named selection of Steps a Flow can plan with",
		pathTool(s.httpGet, "/engine/spaces"),
	)
	srv.Tool("get_space", "Fetch one Space with its selector",
		idTool(s.httpGet, "/engine/spaces/", ""))
	srv.Tool("list_space_steps", "List the Steps a Space selects",
		idTool(s.httpGet, "/engine/spaces/", "/steps"))
	srv.Tool("unregister_space", "Remove a Space",
		idTool(s.httpDelete, "/engine/spaces/", ""))
	srv.Tool(
		"preview_space",
		"Show which Steps a Space would select without saving it. "+spaceShape,
		func(_ *server.Context, args spaceInput) (any, error) {
			if len(args.Space) == 0 {
				return nil, errInvalidParams("space body is required")
			}
			return toolResult(s.httpPost("/engine/spaces/preview", args.Space))
		},
	)
	srv.Tool(
		"register_space",
		"Register a new Space. "+spaceShape,
		func(_ *server.Context, args spaceInput) (any, error) {
			if len(args.Space) == 0 {
				return nil, errInvalidParams("space body is required")
			}
			return toolResult(s.httpPost("/engine/spaces", args.Space))
		},
	)
	srv.Tool(
		"update_space",
		"Replace an existing Space. Fetch it with get_space, change it, and "+
			"send the whole Space",
		func(_ *server.Context, args updateSpaceInput) (any, error) {
			if args.ID == "" || len(args.Space) == 0 {
				return nil, errInvalidParams("id and space body are required")
			}
			return toolResult(s.httpPut("/engine/spaces/"+args.ID, args.Space))
		},
	)
}

func (s *Server) registerFeedTools(srv server.Server) {
	srv.Tool(
		"list_feeds",
		"List Feeds. A Feed is a catalog-wide value folded from settled Step "+
			"outputs",
		pathTool(s.httpGet, "/engine/feeds"),
	)
	srv.Tool("get_feed", "Fetch one Feed with its current value",
		idTool(s.httpGet, "/engine/feeds/", ""))
	srv.Tool("unregister_feed", "Remove a Feed",
		idTool(s.httpDelete, "/engine/feeds/", ""))
	srv.Tool(
		"register_feed",
		"Register a new Feed. "+feedShape,
		func(_ *server.Context, args feedInput) (any, error) {
			if len(args.Feed) == 0 {
				return nil, errInvalidParams("feed body is required")
			}
			return toolResult(s.httpPost("/engine/feeds", args.Feed))
		},
	)
	srv.Tool(
		"update_feed",
		"Replace an existing Feed. Fetch it with get_feed, change it, and "+
			"send the whole Feed",
		func(_ *server.Context, args updateFeedInput) (any, error) {
			if args.ID == "" || len(args.Feed) == 0 {
				return nil, errInvalidParams("id and feed body are required")
			}
			return toolResult(s.httpPut("/engine/feeds/"+args.ID, args.Feed))
		},
	)
}
