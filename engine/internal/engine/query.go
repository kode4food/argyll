package engine

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	flowItem struct {
		id     api.FlowID
		digest *api.FlowDigest
		group  int
		recent int64
	}

	flowStatusEntry struct {
		id        api.FlowID
		status    api.FlowStatus
		timestamp int64
	}

	queryStatus struct {
		indexStatus string
		flowStatus  api.FlowStatus
	}

	flowQueryCursor struct {
		Group  int        `json:"group"`
		Recent int64      `json:"recent"`
		ID     api.FlowID `json:"id"`
	}
)

var (
	ErrInvalidFlowCursor = errors.New("invalid flow cursor")
	ErrQueryFlows        = errors.New("failed to query flows")
)

// ListFlows returns summary information for flows using the query path
func (e *Engine) ListFlows() ([]*api.QueryFlowsItem, error) {
	resp, err := e.QueryFlows(&api.QueryFlowsRequest{
		Sort: api.FlowSortRecentDesc,
	})
	if err != nil {
		return nil, err
	}
	return resp.Flows, nil
}

// QueryFlows returns summary information for flows with filtering and paging
func (e *Engine) QueryFlows(
	req *api.QueryFlowsRequest,
) (*api.QueryFlowsResponse, error) {
	sortOrder := querySortOrder(req)
	items, err := e.buildFlowQueryItems(req)
	if err != nil {
		return nil, err
	}
	sortFlowItems(items, sortOrder)

	start, err := flowStart(items, req.Cursor, sortOrder)
	if err != nil {
		return nil, err
	}

	page, hasMore, nextCursor := paginateFlowItems(items, start, req.Limit)

	return buildFlowQueryResponse(page, len(items), hasMore, nextCursor), nil
}

func querySortOrder(req *api.QueryFlowsRequest) api.FlowSort {
	if req.Sort == "" {
		return api.FlowSortRecentAsc
	}
	return req.Sort
}

// collectRootFlowEntries returns indexed root flow entries
func (e *Engine) collectRootFlowEntries(
	statuses []api.FlowStatus,
) ([]flowStatusEntry, error) {
	entries := []flowStatusEntry{}
	seen := util.Set[api.FlowID]{}

	for _, item := range queryStatuses(statuses) {
		group, err := e.listIndexedEntries(item.indexStatus, item.flowStatus)
		if err != nil {
			return nil, err
		}
		for _, entry := range group {
			if isChildFlowID(entry.id) || seen.Contains(entry.id) {
				continue
			}
			seen.Add(entry.id)
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (e *Engine) listIndexedEntries(
	status string, flowStatus api.FlowStatus,
) ([]flowStatusEntry, error) {
	store := e.flowExec.GetStore()
	entries, err := store.ListAggregatesByStatus(e.ctx, status)
	if err != nil {
		return nil, err
	}

	res := make([]flowStatusEntry, 0, len(entries))
	for _, entry := range entries {
		flowID, ok := events.ParseFlowID(entry.ID)
		if !ok {
			return nil, fmt.Errorf("%w: invalid flow status entry %s",
				ErrQueryFlows, entry.ID.Join(":"))
		}
		res = append(res, flowStatusEntry{
			id:        flowID,
			status:    flowStatus,
			timestamp: entry.Timestamp.UnixNano(),
		})
	}
	return res, nil
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

// flowQueryOrdering returns the grouping and recent timestamp for a digest
func flowQueryOrdering(digest *api.FlowDigest, recent int64) (int, int64) {
	group := 1
	if digest.Status == api.FlowActive {
		group = 0
		return group, recent
	}
	return group, recent
}

// buildFlowQueryItems converts indexed flow entries into sortable query items
func (e *Engine) buildFlowQueryItems(
	req *api.QueryFlowsRequest,
) ([]flowItem, error) {
	entries, err := e.collectRootFlowEntries(req.Statuses)
	if err != nil {
		return nil, err
	}

	items := make([]flowItem, 0, len(entries))
	for _, entry := range entries {
		if req.IDPrefix != "" &&
			!strings.HasPrefix(string(entry.id), req.IDPrefix) {
			continue
		}
		item, ok, err := e.buildFlowQueryItem(req, entry)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (e *Engine) buildFlowQueryItem(
	req *api.QueryFlowsRequest, entry flowStatusEntry,
) (flowItem, bool, error) {
	if !queryNeedsFlow(req) {
		return flowItemFromEntry(entry), true, nil
	}

	flow, err := e.loadIndexedFlow(entry.id)
	if err != nil {
		return flowItem{}, false, err
	}
	if flow == nil {
		return flowItem{}, false, nil
	}

	item := flowItemFromEntry(entry)
	digest := flowDigest(flow)
	if !labelsMatch(digest.Labels, req.Labels) {
		return flowItem{}, false, nil
	}
	item.digest = digest
	return item, true, nil
}

func (e *Engine) loadIndexedFlow(
	flowID api.FlowID,
) (*api.FlowState, error) {
	flow, err := e.GetFlowState(flowID)
	if err == nil {
		return flow, nil
	}
	if !errors.Is(err, ErrFlowNotFound) {
		return nil, err
	}
	return nil, nil
}

func isChildFlowID(id api.FlowID) bool {
	return strings.ContainsRune(string(id), ':')
}

func queryNeedsFlow(req *api.QueryFlowsRequest) bool {
	return len(req.Labels) > 0
}

func flowDigest(flow *api.FlowState) *api.FlowDigest {
	return &api.FlowDigest{
		Status:      flow.Status,
		CreatedAt:   flow.CreatedAt,
		CompletedAt: flow.CompletedAt,
		Labels:      flow.Labels,
		Error:       flow.Error,
	}
}

func flowDigestFromEntry(entry flowStatusEntry) *api.FlowDigest {
	ts := time.Unix(0, entry.timestamp).UTC()
	digest := &api.FlowDigest{
		Status:    entry.status,
		CreatedAt: ts,
	}
	if entry.status != api.FlowActive {
		digest.CompletedAt = ts
	}
	return digest
}

func flowItemFromEntry(entry flowStatusEntry) flowItem {
	digest := flowDigestFromEntry(entry)
	group, recent := flowQueryOrdering(digest, entry.timestamp)
	return flowItem{
		id:     entry.id,
		digest: digest,
		group:  group,
		recent: recent,
	}
}

func queryStatuses(statuses []api.FlowStatus) []queryStatus {
	if len(statuses) == 0 {
		return []queryStatus{
			{indexStatus: events.FlowStatusActive, flowStatus: api.FlowActive},
			{
				indexStatus: events.FlowStatusCompleted,
				flowStatus:  api.FlowCompleted,
			},
			{indexStatus: events.FlowStatusFailed, flowStatus: api.FlowFailed},
		}
	}

	res := make([]queryStatus, 0, len(statuses))
	seen := util.Set[api.FlowStatus]{}
	for _, status := range statuses {
		if seen.Contains(status) {
			continue
		}
		seen.Add(status)
		switch status {
		case api.FlowActive:
			res = append(res, queryStatus{
				indexStatus: events.FlowStatusActive,
				flowStatus:  api.FlowActive,
			})
		case api.FlowCompleted:
			res = append(res, queryStatus{
				indexStatus: events.FlowStatusCompleted,
				flowStatus:  api.FlowCompleted,
			})
		case api.FlowFailed:
			res = append(res, queryStatus{
				indexStatus: events.FlowStatusFailed,
				flowStatus:  api.FlowFailed,
			})
		}
	}
	return res
}

func flowLess(left, right flowItem, sortOrder api.FlowSort) bool {
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

func sortFlowItems(items []flowItem, sortOrder api.FlowSort) {
	sort.Slice(items, func(i, j int) bool {
		return flowLess(items[i], items[j], sortOrder)
	})
}

func flowLessKey(
	cursor flowQueryCursor, item flowItem, sortOrder api.FlowSort,
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

func flowStart(
	items []flowItem, cursorValue string, sortOrder api.FlowSort,
) (int, error) {
	if cursorValue == "" {
		return 0, nil
	}
	cursor, err := decodeFlowQueryCursor(cursorValue)
	if err != nil {
		return 0, err
	}
	for i, item := range items {
		if flowLessKey(cursor, item, sortOrder) {
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

func paginateFlowItems(
	items []flowItem, start, limit int,
) ([]flowItem, bool, string) {
	end := len(items)
	if limit > 0 && start+limit < end {
		end = start + limit
	}

	page := items
	if start < len(items) {
		page = items[start:end]
	} else {
		page = []flowItem{}
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
	page []flowItem, total int, hasMore bool, nextCursor string,
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
