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
	reply, err := w.converseChat(ctx, &req, usage, servers)
	if err != nil {
		slog.Warn("Agent chat failed", slog.String("provider", string(req.Provider)),
			slog.String("model", string(req.Model)), log.Error(err))
		p := problemOf(err)
		rw.Header().Set("Content-Type", api.ProblemJSONContentType)
		rw.WriteHeader(p.Status)
		_ = json.NewEncoder(rw).Encode(p)
		return
	}
	rw.Header().Set("Content-Type", api.JSONContentType)
	_ = json.NewEncoder(rw).Encode(contract.AgentChatReply{
		Reply: reply,
		Usage: usage,
	})
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
) (string, error) {
	cfg := &api.AgentConfig{
		MCP:       []api.MCPServer{{Server: ChatServer}},
		Provider:  req.Provider,
		Model:     req.Model,
		MaxTurns:  chatMaxTurns,
		MaxTokens: chatMaxTokens,
	}
	box, err := openToolbox(ctx, cfg.MCP, servers)
	if err != nil {
		return "", err
	}
	defer box.close()

	user, err := json.Marshal(req.Messages)
	if err != nil {
		return "", err
	}
	conv, err := w.provider.Open(&Chat{
		Config:        cfg,
		System:        strings.TrimSpace(chatPrompt) + " " + strings.TrimSpace(chatStyle),
		CompactSystem: strings.TrimSpace(compactChatPrompt) + " " + strings.TrimSpace(chatStyle),
		User:          string(user),
		Tools:         append(box.specs, replySpec()),
	})
	if err != nil {
		return "", err
	}
	return runTurns(ctx, w, conv, runTurnsArgs[string]{
		cfg:   cfg,
		usage: usage,
		nudge: strings.TrimSpace(replyNudge),
		handle: func(call ToolCall) (string, string, bool, error) {
			if text, ok := replyText(call); ok {
				return "", text, true, nil
			}
			reply, err := chatToolReply(ctx, box, call)
			return reply, "", false, err
		},
	})
}

func replyText(call ToolCall) (string, bool) {
	if call.Name != replyTool {
		return "", false
	}
	var args struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(call.Args, &args); err != nil || args.Text == "" {
		return "", false
	}
	return args.Text, true
}

// chatToolReply runs an MCP tool call, answering any mistake with a reply the model can correct
func chatToolReply(ctx context.Context, box *toolbox, call ToolCall) (string, error) {
	if call.Name == replyTool {
		return "Error: reply takes a non-empty text string", nil
	}
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
				"text": map[string]any{"type": "string"},
			},
			"required":             []any{"text"},
			"additionalProperties": false,
		},
	}
}
