package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kode4food/argyll/chat/contract"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

const (
	// ChatServer is the only MCP server offered to a chat
	ChatServer = "argyll"

	chatTimeout   = 2 * time.Minute
	chatMaxTurns  = 20
	chatMaxTokens = 4096
	replyTool     = "reply"
)

var (
	ErrChatServerMissing = errors.New("MCP server " + ChatServer + " not configured")
	ErrReplyArgsInvalid  = errors.New("reply arguments invalid")
	ErrReplyTextEmpty    = errors.New("reply takes a non-empty text string")
)

func (w *Worker) chat(rw http.ResponseWriter, r *http.Request) {
	var req contract.AgentChatRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err == nil {
		err = req.Validate()
	}
	if err != nil {
		writeProblem(rw, http.StatusBadRequest, fmt.Errorf("%w: %w", ErrInvalidRequest, err))
		return
	}
	if !w.provider.Supports(req.Provider) {
		err := fmt.Errorf("%w: %s", ErrUnknownProvider, req.Provider)
		writeProblem(rw, http.StatusUnprocessableEntity, err)
		return
	}
	if _, ok := w.cfg.Servers[ChatServer]; !ok {
		writeProblem(rw, http.StatusUnprocessableEntity, ErrChatServerMissing)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), chatTimeout)
	defer cancel()
	usage := api.Usage{
		usageInput:  0,
		usageOutput: 0,
	}
	servers := w.chatServers(r.Header.Get("Authorization"))
	res, err := w.converseChat(ctx, &req, usage, servers)
	if err != nil {
		slog.Warn("Agent chat failed", slog.String("provider", string(req.Provider)),
			slog.String("model", string(req.Model)), log.Error(err))
		p := problemOf(err)
		// The caller passed auth before this point, so a 401 or 403 here is the worker's own
		// provider credentials, and passing it on would sign the caller out
		if p.Status == http.StatusUnauthorized || p.Status == http.StatusForbidden {
			p = util.MutableCopy(p)
			p.Status = http.StatusBadGateway
		}
		rw.Header().Set("Content-Type", api.ProblemJSONContentType)
		rw.WriteHeader(p.Status)
		_ = json.NewEncoder(rw).Encode(p)
		return
	}
	res.Usage = usage
	rw.Header().Set("Content-Type", api.JSONContentType)
	_ = json.NewEncoder(rw).Encode(res)
}

// chatServers offers the Argyll MCP server with only the chat caller's Authorization, so its tools
// act as that caller and never with the worker's own credentials
func (w *Worker) chatServers(auth string) map[string]MCPServer {
	s := w.cfg.Servers[ChatServer]
	headers := map[string]string{}
	if auth != "" {
		headers["Authorization"] = auth
	}
	return map[string]MCPServer{
		ChatServer: {
			Headers: headers,
			URL:     s.URL,
		},
	}
}

func (w *Worker) converseChat(
	ctx context.Context, req *contract.AgentChatRequest, usage api.Usage,
	servers map[string]MCPServer,
) (contract.AgentChatReply, error) {
	cfg := &api.AgentConfig{
		MCP:       []api.MCPServer{{Server: ChatServer}},
		Provider:  req.Provider,
		Model:     req.Model,
		MaxTurns:  chatMaxTurns,
		MaxTokens: chatMaxTokens,
	}
	box, err := openToolbox(ctx, cfg.MCP, servers)
	if err != nil {
		return contract.AgentChatReply{}, err
	}
	defer box.close()

	user, err := json.Marshal(struct {
		View     *contract.ChatView     `json:"view,omitempty"`
		Messages []contract.ChatMessage `json:"messages"`
	}{View: req.View, Messages: req.Messages})
	if err != nil {
		return contract.AgentChatReply{}, err
	}
	conv, err := w.provider.Open(&Chat{
		Config: cfg,
		System: strings.TrimSpace(chatPrompt) + " " + strings.TrimSpace(chatStyle),
		User:   string(user),
		Tools:  append(box.specs, replySpec()),
	})
	if err != nil {
		return contract.AgentChatReply{}, err
	}
	return runTurns(ctx, w, conv, runTurnsArgs[contract.AgentChatReply]{
		cfg:   cfg,
		usage: usage,
		nudge: strings.TrimSpace(replyNudge),
		handle: func(call ToolCall) (string, contract.AgentChatReply, bool, error) {
			if call.Name != replyTool {
				reply, err := chatToolReply(ctx, box, call)
				return reply, contract.AgentChatReply{}, false, err
			}
			res, err := replyOf(call)
			if err != nil {
				return "Error: " + err.Error(), contract.AgentChatReply{}, false, nil
			}
			return "", res, true, nil
		},
	})
}

func replyOf(call ToolCall) (contract.AgentChatReply, error) {
	var args struct {
		Action *contract.ChatAction `json:"action"`
		Text   string               `json:"text"`
	}
	if err := json.Unmarshal(call.Args, &args); err != nil {
		return contract.AgentChatReply{}, fmt.Errorf("%w: %w", ErrReplyArgsInvalid, err)
	}
	if args.Text == "" {
		return contract.AgentChatReply{}, ErrReplyTextEmpty
	}
	res := contract.AgentChatReply{
		Action: args.Action,
		Reply:  args.Text,
	}
	if args.Action == nil {
		return res, nil
	}
	return res, args.Action.Validate()
}

// chatToolReply runs an MCP tool call, answering any mistake with a reply the model can correct
func chatToolReply(ctx context.Context, box *toolbox, call ToolCall) (string, error) {
	var args api.Args
	if err := json.Unmarshal(call.Args, &args); err != nil || args == nil {
		return "Error: arguments must be a JSON object", nil
	}
	if !box.has(call.Name) {
		return "Error: unknown tool " + call.Name, nil
	}
	return box.call(ctx, call.Name, args)
}

func replySpec() ToolSpec {
	return ToolSpec{
		Name:        replyTool,
		Description: "Send your answer to the user and end the turn",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text":   map[string]any{"type": "string"},
				"action": actionSchema(),
			},
			"required":             []any{"text"},
			"additionalProperties": false,
		},
	}
}

func actionSchema() map[string]any {
	id := map[string]any{"type": "string"}
	return map[string]any{
		"type":        "object",
		"description": "Take the web UI to what the reply is about. It only navigates",
		"properties": map[string]any{
			"type": map[string]any{
				"type": "string",
				"enum": []any{
					contract.ChatActionOpenOverview,
					contract.ChatActionOpenFlow,
					contract.ChatActionSelectGoals,
					contract.ChatActionOpenStep,
					contract.ChatActionOpenSpace,
					contract.ChatActionOpenFeed,
				},
				"description": "open_overview shows the overview, in space_id when set. " +
					"open_flow shows the Flow flow_id. " +
					"select_goals selects goals in the new Flow form. " +
					"open_step opens the Step step_id in the Step editor. " +
					"open_space opens the Space manager, at space_id when set. " +
					"open_feed opens the Feed manager, at feed_id when set",
			},
			"space_id": id,
			"flow_id":  id,
			"step_id":  id,
			"feed_id":  id,
			"goals": map[string]any{
				"type":        "object",
				"description": "Goals as start_flow takes them",
			},
		},
		"required":             []any{"type"},
		"additionalProperties": false,
	}
}
