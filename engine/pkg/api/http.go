package api

import (
	"fmt"
	"mime"
	"net/http"
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

	// ActionCompensate reports the outcome of a step compensation
	ActionCompensate CallbackAction = "compensate"

	// CallbackRoute is the router pattern serving step callbacks. Its
	// parameters and CallbackPath describe the same URL
	CallbackRoute = "/callbacks/:flow_id/:step_id/:token/:action"
	ParamFlowID   = "flow_id"
	ParamStepID   = "step_id"
	ParamToken    = "token"
	ParamAction   = "action"

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

// CallbackPath builds the path an async handler posts its outcome to
func CallbackPath(
	flowID FlowID, stepID StepID, tkn Token, action CallbackAction,
) string {
	return fmt.Sprintf("/callbacks/%s/%s/%s/%s", flowID, stepID, tkn, action)
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
	addHeaderMeta(meta, MetaFlowID, header.Get(HeaderFlowID))
	addHeaderMeta(meta, MetaStepID, header.Get(HeaderStepID))
	addHeaderMeta(meta, MetaReceiptToken, header.Get(HeaderReceiptToken))
	addHeaderMeta(meta, MetaWebhookURL, header.Get(HeaderWebhookURL))
	return meta
}

func setMetaHeader(header http.Header, name string, meta Metadata, key string) {
	if value, ok := meta.GetString[string](key); ok {
		header.Set(name, value)
	}
}

func addHeaderMeta(meta Metadata, key, value string) {
	if value != "" {
		meta[key] = value
	}
}
