package engine

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	flowQueryItem struct {
		id     api.FlowID
		digest *api.FlowDigest
		group  int
		recent int64
	}

	flowQueryCursor struct {
		Group  int        `json:"group"`
		Recent int64      `json:"recent"`
		ID     api.FlowID `json:"id"`
	}

	flowQueryFilter func(api.FlowID, *api.FlowDigest) bool
)

// ListFlows returns summary information for active and deactivated flows
func (e *Engine) ListFlows() ([]*api.QueryFlowsItem, error) {
	resp, err := e.QueryFlows(nil)
	if err != nil {
		return nil, err
	}
	return resp.Flows, nil
}

// QueryFlows returns summary information for flows with filtering and paging
func (e *Engine) QueryFlows(
	req *api.QueryFlowsRequest,
) (*api.QueryFlowsResponse, error) {
	engState, err := e.GetEngineState()
	if err != nil {
		return nil, err
	}

	req = normalizeQueryFlowsRequest(req)
	sortOrder := querySortOrder(req)
	flowIDs := collectRootFlowIDs(engState)
	filters := buildFlowQueryFilters(req)
	items := buildFlowQueryItems(engState, flowIDs, filters)
	sortFlowQueryItems(items, sortOrder)

	start, err := queryFlowStart(items, req, sortOrder)
	if err != nil {
		return nil, err
	}

	page, hasMore, nextCursor := paginateFlowQuery(
		items, start, req.Limit,
	)

	return buildFlowQueryResponse(page, len(items), hasMore, nextCursor), nil
}

func normalizeQueryFlowsRequest(
	req *api.QueryFlowsRequest,
) *api.QueryFlowsRequest {
	if req == nil {
		return &api.QueryFlowsRequest{}
	}
	return req
}

func querySortOrder(req *api.QueryFlowsRequest) api.FlowSort {
	if req.Sort == "" {
		return api.FlowSortRecentAsc
	}
	return req.Sort
}

// collectRootFlowIDs returns active & deactivated flow IDs excluding children
func collectRootFlowIDs(engState *api.EngineState) []api.FlowID {
	count := len(engState.Active) + len(engState.Deactivated)
	flowIDs := make([]api.FlowID, 0, count)
	seen := util.Set[api.FlowID]{}
	for id, info := range engState.Active {
		if info != nil && info.ParentFlowID != "" {
			continue
		}
		seen.Add(id)
		flowIDs = append(flowIDs, id)
	}
	for _, info := range engState.Deactivated {
		if info.ParentFlowID != "" {
			continue
		}
		if seen.Contains(info.FlowID) {
			continue
		}
		seen.Add(info.FlowID)
		flowIDs = append(flowIDs, info.FlowID)
	}
	return flowIDs
}

func labelsMatch(flowLabels, queryLabels api.Labels) bool {
	if len(queryLabels) == 0 {
		return true
	}
	if len(flowLabels) == 0 {
		return false
	}
	for key, value := range queryLabels {
		if flowLabels[key] != value {
			return false
		}
	}
	return true
}

// buildFlowQueryFilters assembles flow filters from the query request
func buildFlowQueryFilters(req *api.QueryFlowsRequest) []flowQueryFilter {
	filters := make([]flowQueryFilter, 0, 3)
	if req.IDPrefix != "" {
		filters = append(filters, func(id api.FlowID, _ *api.FlowDigest) bool {
			return strings.HasPrefix(string(id), req.IDPrefix)
		})
	}
	if len(req.Statuses) > 0 {
		statusFilter := util.Set[api.FlowStatus]{}
		for _, s := range req.Statuses {
			statusFilter.Add(s)
		}
		filters = append(filters,
			func(_ api.FlowID, digest *api.FlowDigest) bool {
				return statusFilter.Contains(digest.Status)
			},
		)
	}
	if len(req.Labels) > 0 {
		filters = append(filters,
			func(_ api.FlowID, digest *api.FlowDigest) bool {
				return labelsMatch(digest.Labels, req.Labels)
			},
		)
	}
	return filters
}

// matchesFlowQueryFilters returns true when all filters accept the flow digest
func matchesFlowQueryFilters(
	flowID api.FlowID, digest *api.FlowDigest, filters []flowQueryFilter,
) bool {
	for _, filter := range filters {
		if !filter(flowID, digest) {
			return false
		}
	}
	return true
}

// flowQueryOrdering returns the grouping and recent timestamp for a digest
func flowQueryOrdering(digest *api.FlowDigest) (int, int64) {
	group := 1
	recent := digest.CreatedAt.UnixNano()
	if digest.Status == api.FlowActive {
		group = 0
		return group, recent
	}
	if !digest.CompletedAt.IsZero() {
		recent = digest.CompletedAt.UnixNano()
	}
	return group, recent
}

// buildFlowQueryItems converts flow digests into sortable query items
func buildFlowQueryItems(
	engState *api.EngineState, flowIDs []api.FlowID, filters []flowQueryFilter,
) []flowQueryItem {
	items := make([]flowQueryItem, 0, len(flowIDs))
	for _, flowID := range flowIDs {
		digest, ok := engState.FlowDigests[flowID]
		if !ok || digest == nil {
			continue
		}
		if !matchesFlowQueryFilters(flowID, digest, filters) {
			continue
		}
		group, recent := flowQueryOrdering(digest)
		items = append(items, flowQueryItem{
			id:     flowID,
			digest: digest,
			group:  group,
			recent: recent,
		})
	}
	return items
}

func flowQueryLess(
	left flowQueryItem, right flowQueryItem, sortOrder api.FlowSort,
) bool {
	if left.group != right.group {
		return left.group < right.group
	}
	if left.recent != right.recent {
		if sortOrder == api.FlowSortRecentAsc {
			return left.recent < right.recent
		}
		return left.recent > right.recent
	}
	return left.id < right.id
}

func sortFlowQueryItems(items []flowQueryItem, sortOrder api.FlowSort) {
	sort.Slice(items, func(i, j int) bool {
		return flowQueryLess(items[i], items[j], sortOrder)
	})
}

func flowQueryLessKey(
	cursor flowQueryCursor, item flowQueryItem, sortOrder api.FlowSort,
) bool {
	if cursor.Group != item.group {
		return cursor.Group < item.group
	}
	if cursor.Recent != item.recent {
		if sortOrder == api.FlowSortRecentAsc {
			return cursor.Recent < item.recent
		}
		return cursor.Recent > item.recent
	}
	return cursor.ID < item.id
}

// decodeFlowQueryCursor parses a cursor string into a cursor key
func decodeFlowQueryCursor(value string) (flowQueryCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return flowQueryCursor{},
			fmt.Errorf("%w: %s", ErrInvalidFlowCursor, err)
	}
	var cursor flowQueryCursor
	if err := json.Unmarshal(b, &cursor); err != nil {
		return flowQueryCursor{},
			fmt.Errorf("%w: %s", ErrInvalidFlowCursor, err)
	}
	return cursor, nil
}

// queryFlowStart finds the first item after the cursor for pagination
func queryFlowStart(
	items []flowQueryItem, req *api.QueryFlowsRequest, sortOrder api.FlowSort,
) (int, error) {
	if req.Cursor == "" {
		return 0, nil
	}
	cursor, err := decodeFlowQueryCursor(req.Cursor)
	if err != nil {
		return 0, err
	}
	for i, item := range items {
		if flowQueryLessKey(cursor, item, sortOrder) {
			return i, nil
		}
	}
	return len(items), nil
}

func encodeFlowQueryCursor(cursor flowQueryCursor) string {
	b, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// paginateFlowQuery slices the item list and returns the next cursor if needed
func paginateFlowQuery(
	items []flowQueryItem, start, limit int,
) ([]flowQueryItem, bool, string) {
	end := len(items)
	if limit > 0 && start+limit < end {
		end = start + limit
	}

	page := items
	if start < len(items) {
		page = items[start:end]
	} else {
		page = []flowQueryItem{}
	}

	hasMore := end < len(items)
	if !hasMore || len(page) == 0 {
		return page, hasMore, ""
	}

	last := page[len(page)-1]
	nextCursor := encodeFlowQueryCursor(flowQueryCursor{
		Group:  last.group,
		Recent: last.recent,
		ID:     last.id,
	})
	return page, hasMore, nextCursor
}

// buildFlowQueryResponse converts items into the response payload
func buildFlowQueryResponse(
	page []flowQueryItem, total int, hasMore bool, nextCursor string,
) *api.QueryFlowsResponse {
	flows := make([]*api.QueryFlowsItem, 0, len(page))
	for _, item := range page {
		flows = append(flows, &api.QueryFlowsItem{
			ID:     item.id,
			Digest: item.digest,
		})
	}
	return &api.QueryFlowsResponse{
		Flows:      flows,
		Count:      len(flows),
		Total:      total,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}
}
