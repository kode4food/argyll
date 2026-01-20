package api

// Metadata contains additional context passed to step handlers
type Metadata map[string]any

const (
	MetaFlowID       = "flow_id"
	MetaStepID       = "step_id"
	MetaReceiptToken = "receipt_token"
	MetaWebhookURL   = "webhook_url"

	MetaParentFlowID        = "parent_flow_id"
	MetaParentStepID        = "parent_step_id"
	MetaParentWorkItemToken = "parent_work_item_token"
)

func GetMetaString[T ~string](meta Metadata, key string) (T, bool) {
	var zero T
	val, ok := meta[key]
	if !ok {
		return zero, false
	}

	switch v := val.(type) {
	case T:
		if v == "" {
			return zero, false
		}
		return v, true
	case string:
		if v == "" {
			return zero, false
		}
		return T(v), true
	default:
		return zero, false
	}
}
