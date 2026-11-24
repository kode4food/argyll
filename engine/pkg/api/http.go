package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// ProblemDetails describes an RFC 9457 problem details response body
type ProblemDetails struct {
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	Status   int    `json:"status,omitempty"`
}

// CallbackAction names the step action whose outcome a callback reports
type CallbackAction string

const (
	// ActionInvoke reports the outcome of a step invocation
	ActionInvoke CallbackAction = "invoke"

	// ActionAgent reports an Agent invocation's outcome envelope
	ActionAgent CallbackAction = "agent"

	// ActionCompensate reports the outcome of a step compensation
	ActionCompensate CallbackAction = "compensate"

	// CallbackRoute is the router pattern serving step callbacks. Its parameters and CallbackPath
	// describe the same URL
	CallbackRoute = "/callbacks/:flow_id/:step_id/:token/:action/:secret"
	ParamFlowID   = "flow_id"
	ParamStepID   = "step_id"
	ParamToken    = "token"
	ParamAction   = "action"
	ParamSecret   = "secret"

	// HeaderFlowID carries the executing flow ID on HTTP step requests
	HeaderFlowID = "Argyll-Flow-ID"

	// HeaderStepID carries the executing step ID on HTTP step requests
	HeaderStepID = "Argyll-Step-ID"

	// HeaderReceiptToken carries the work-item token for idempotency
	HeaderReceiptToken = "Argyll-Receipt-Token"

	// HeaderWebhookURL carries the async completion URL for async HTTP steps
	HeaderWebhookURL = "Argyll-Webhook-URL"

	// JSONContentType is the standard JSON media type
	JSONContentType = "application/json"

	// ProblemJSONContentType is the RFC 9457 JSON problem details media type
	ProblemJSONContentType = "application/problem+json"
)

const (
	maxProblemBody = 4 * 1024 // 4 KB

	// MaxHTTPRedirects matches the limit http.Client applies by default
	MaxHTTPRedirects = 10
)

var (
	ErrCrossOriginRedirect = errors.New("redirect to another origin refused")
	ErrTooManyRedirects    = errors.New("too many redirects")
)

// CallbackPath builds the secret-bearing path for an async Step callback
func CallbackPath(fs FlowStep, tkn Token, action CallbackAction, secret string) string {
	return fmt.Sprintf("/callbacks/%s/%s/%s/%s/%s", fs.FlowID, fs.StepID, tkn, action, secret)
}

// RedactCallback drops the secret from a callback URL so it can be logged
func RedactCallback(callback string) string {
	return callback[:strings.LastIndexByte(callback, '/')+1]
}

// SameOriginRedirect limits an HTTP redirect chain to the original origin
func SameOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= MaxHTTPRedirects {
		return ErrTooManyRedirects
	}
	if req.URL.Scheme != via[0].URL.Scheme || req.URL.Host != via[0].URL.Host {
		return fmt.Errorf("%w: %s", ErrCrossOriginRedirect, req.URL.Host)
	}
	return nil
}

// WithoutURL unwraps an HTTP client error to omit the request URL
func WithoutURL(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return ue.Err
	}
	return err
}

// CallbackHash hashes a callback secret for its invocation or compensation phase
func CallbackHash(action CallbackAction, secret string) string {
	if action == ActionAgent {
		action = ActionInvoke
	}
	sum := sha256.Sum256([]byte(string(action) + "/" + secret))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// NewProblem creates a problem details response with standard title text
func NewProblem(status int, detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
}

// ReadHTTPProblem preserves a bounded response body and its actual HTTP status
func ReadHTTPProblem(res *http.Response) *ProblemDetails {
	body, err := io.ReadAll(io.LimitReader(res.Body, maxProblemBody))
	p := NewProblem(res.StatusCode, strings.TrimSpace(string(body)))
	if err != nil {
		p.Detail = fmt.Sprintf("%s: %s", p.Detail, err)
	}
	var remote ProblemDetails
	if err == nil && json.Unmarshal(body, &remote) == nil && remote.Error() != "" {
		p = &remote
		p.Status = res.StatusCode
	}
	if p.Instance == "" && res.Request != nil && res.Request.URL != nil {
		u := res.Request.URL
		p.Instance = (&url.URL{
			Scheme: u.Scheme,
			Host:   u.Host,
			Path:   u.Path,
		}).String()
	}
	return p
}

// IsProblemJSON reports whether the content type is Problem Details JSON
func IsProblemJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == ProblemJSONContentType
}

// SetMetadataHeaders maps Argyll metadata values onto HTTP headers
func SetMetadataHeaders(header http.Header, meta Metadata) {
	setMetaHeader(header, HeaderFlowID, meta, MetaFlowID)
	setMetaHeader(header, HeaderStepID, meta, MetaStepID)
	setMetaHeader(header, HeaderReceiptToken, meta, MetaReceiptToken)
	setMetaHeader(header, HeaderWebhookURL, meta, MetaWebhookURL)
}

// MetadataFromHeaders maps Argyll HTTP headers into metadata values
func MetadataFromHeaders(header http.Header) Metadata {
	meta := Metadata{}
	addHeaderMeta(meta, headerMetaArgs{
		key:   MetaFlowID,
		value: header.Get(HeaderFlowID),
	})
	addHeaderMeta(meta, headerMetaArgs{
		key:   MetaStepID,
		value: header.Get(HeaderStepID),
	})
	addHeaderMeta(meta, headerMetaArgs{
		key:   MetaReceiptToken,
		value: header.Get(HeaderReceiptToken),
	})
	addHeaderMeta(meta, headerMetaArgs{
		key:   MetaWebhookURL,
		value: header.Get(HeaderWebhookURL),
	})
	return meta
}

// Error returns the most useful human-readable problem details message
func (p *ProblemDetails) Error() string {
	if p == nil {
		return ""
	}
	if p.Detail != "" {
		return p.Detail
	}
	if p.Title != "" {
		return p.Title
	}
	return http.StatusText(p.Status)
}

func setMetaHeader(header http.Header, name string, meta Metadata, key string) {
	if value, ok := meta.GetString[string](key); ok {
		header.Set(name, value)
	}
}

type headerMetaArgs struct {
	key   string
	value string
}

func addHeaderMeta(meta Metadata, args headerMetaArgs) {
	if args.value != "" {
		meta[args.key] = args.value
	}
}
