package engine

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type (
	// flowQueryCursor records, per status, the last flow a prior page took from that status
	flowQueryCursor map[api.FlowStatus]flowPosition

	flowPosition struct {
		ID     api.FlowID `json:"id"`
		Recent int64      `json:"recent"`
	}
)

// DefaultFlowQueryLimit is the page size a query uses by default
const DefaultFlowQueryLimit = 100

var (
	ErrInvalidFlowCursor = errors.New("invalid flow cursor")
	ErrQueryFlows        = errors.New("failed to query flows")
)

var allQueryStatuses = []api.FlowStatus{api.FlowActive, api.FlowCompleted, api.FlowFailed}

// QueryFlows returns one page of root flow summaries, newest first across the requested statuses.
// Each status gets one index read of up to limit entries, resuming from its own cursor position,
// so a request's cost stays bounded for any filter. HasMore reports more to read, even when
// filtering shortens a page
func (e *Engine) QueryFlows(req *api.QueryFlowsRequest) (*api.QueryFlowsResponse, error) {
	cursor, err := decodeFlowQueryCursor(req.Cursor)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit == 0 {
		limit = DefaultFlowQueryLimit
	}

	next := maps.Clone(cursor)
	flows := []*api.QueryFlowsItem{}
	more := false
	for _, status := range queryStatuses(req.Statuses) {
		entries, err := e.flowStore.ListAggregatesByStatus(timebox.StatusQuery{
			Status: string(status),
			Type:   events.FlowPrefix,
			After:  cursor.after(status),
			Limit:  limit,
		})
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			continue
		}
		more = more || len(entries) == limit
		// The cursor moves past every entry read, filtered or kept
		last := entries[len(entries)-1]
		next[status] = flowPosition{
			ID:     api.FlowID(last.ID.Key),
			Recent: last.Timestamp.UnixNano(),
		}
		page, err := e.admitFlows(req, status, entries)
		if err != nil {
			return nil, err
		}
		flows = append(flows, page...)
	}

	// A stable sort keeps each status's own order, so a page takes a prefix of every status
	slices.SortStableFunc(flows, func(l, r *api.QueryFlowsItem) int {
		return r.Timestamp.Compare(l.Timestamp)
	})
	if len(flows) > limit {
		more = true
		flows = cursor.trim(next, flows, limit)
	}

	res := &api.QueryFlowsResponse{
		Flows:   flows,
		Count:   len(flows),
		HasMore: more,
	}
	if more {
		res.NextCursor = encodeFlowQueryCursor(next)
	}
	return res, nil
}

// ListFlows returns one page of all root flow summaries, in QueryFlows order
func (e *Engine) ListFlows(cursor string, limit int) (*api.QueryFlowsResponse, error) {
	return e.QueryFlows(&api.QueryFlowsRequest{
		Cursor: cursor,
		Limit:  limit,
	})
}

// admitFlows keeps the entries matching the request's ID prefix and tags, reading every entry's
// tags in one batch when the request filters by tag
func (e *Engine) admitFlows(
	req *api.QueryFlowsRequest, status api.FlowStatus, entries []timebox.StatusEntry,
) ([]*api.QueryFlowsItem, error) {
	var tags map[timebox.AggregateID][]string
	if len(req.Tags) > 0 {
		ids := make([]timebox.AggregateID, len(entries))
		for i, entry := range entries {
			ids[i] = entry.ID
		}
		var err error
		if tags, err = e.flowStore.GetAggregateTags(ids...); err != nil {
			return nil, err
		}
	}

	var res []*api.QueryFlowsItem
	for _, entry := range entries {
		fid, ok := events.ParseFlowID(entry.ID)
		if !ok {
			return nil, errors.Join(ErrQueryFlows,
				fmt.Errorf("%w: %s", ErrInvalidFlowStatusEntry, entry.ID.String()),
			)
		}
		if !strings.HasPrefix(string(fid), req.IDPrefix) || !hasTags(tags[entry.ID], req.Tags) {
			continue
		}
		res = append(res, &api.QueryFlowsItem{
			ID:        fid,
			Status:    status,
			Timestamp: entry.Timestamp.UTC(),
		})
	}
	return res, nil
}

// after is where a status's next read resumes, or nil to start from the newest
func (c flowQueryCursor) after(status api.FlowStatus) *timebox.StatusEntry {
	pos, ok := c[status]
	if !ok {
		return nil
	}
	return &timebox.StatusEntry{
		Timestamp: time.Unix(0, pos.Recent).UTC(),
		ID:        events.FlowKey(pos.ID),
	}
}

// trim cuts merged flows to the page. A status with flows past the page resumes after its last
// flow in the page, or from its prior position
func (c flowQueryCursor) trim(
	next flowQueryCursor, flows []*api.QueryFlowsItem, limit int,
) []*api.QueryFlowsItem {
	cut := map[api.FlowStatus]bool{}
	for _, fl := range flows[limit:] {
		cut[fl.Status] = true
	}
	for status := range cut {
		if pos, ok := c[status]; ok {
			next[status] = pos
		} else {
			delete(next, status)
		}
	}
	flows = flows[:limit]
	for _, fl := range flows {
		if cut[fl.Status] {
			next[fl.Status] = flowPosition{
				ID:     fl.ID,
				Recent: fl.Timestamp.UnixNano(),
			}
		}
	}
	return flows
}

func hasTags(have, want []string) bool {
	for _, tag := range want {
		if !slices.Contains(have, tag) {
			return false
		}
	}
	return true
}

func queryStatuses(statuses []api.FlowStatus) []api.FlowStatus {
	if len(statuses) == 0 {
		return allQueryStatuses
	}
	var res []api.FlowStatus
	for _, status := range allQueryStatuses {
		if slices.Contains(statuses, status) {
			res = append(res, status)
		}
	}
	return res
}

// decodeFlowQueryCursor parses a cursor string, where an empty one starts from the newest flows
func decodeFlowQueryCursor(value string) (flowQueryCursor, error) {
	if value == "" {
		return flowQueryCursor{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.Join(ErrInvalidFlowCursor, err)
	}
	var cursor flowQueryCursor
	if err := json.Unmarshal(b, &cursor); err != nil {
		return nil, errors.Join(ErrInvalidFlowCursor, err)
	}
	return cursor, nil
}

func encodeFlowQueryCursor(cursor flowQueryCursor) string {
	b, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
