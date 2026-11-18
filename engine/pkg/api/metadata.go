package api

import "reflect"

// Metadata contains additional context passed to step handlers
type Metadata Map[string, any]

const (
	MetaFlowID       = "flow_id"
	MetaStepID       = "step_id"
	MetaReceiptToken = "receipt_token"
	MetaWebhookURL   = "webhook_url"

	MetaParentFlowID        = "parent_flow_id"
	MetaParentStepID        = "parent_step_id"
	MetaParentWorkItemToken = "parent_work_item_token"
)

// Apply will merge the keys/values of the other metadata set into this one
func (m Metadata) Apply(other Metadata) Metadata {
	return Apply(m, other)
}

// GetString retrieves a value of any string type from the metadata as the requested string type,
// returning false if it is missing, empty, or not a string
func (m Metadata) GetString[T ~string](key string) (T, bool) {
	var zero T
	v := reflect.ValueOf(m[key])
	if v.Kind() != reflect.String || v.Len() == 0 {
		return zero, false
	}
	return T(v.String()), true
}
