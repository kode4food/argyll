import { api, FlowSummary } from "../api";
import {
  compareFlows,
  mergeFlowLists,
  toFlowSummary,
} from "./flowStoreHelpers";

export const FLOW_LIST_PAGE_SIZE = 1000;

type FlowsSetFn = (update: {
  flows?: FlowSummary[];
  flowsCursor?: string | null;
  flowsHasMore?: boolean;
  flowsLoading?: boolean;
  error?: string | null;
}) => void;

export async function loadFlowsImpl(set: FlowsSetFn): Promise<void> {
  try {
    set({ flowsLoading: true });
    const resp = await api.listFlowsPage({ limit: FLOW_LIST_PAGE_SIZE });
    const flows = (resp.flows || []).map(toFlowSummary);
    set({
      flows: flows.sort(compareFlows),
      flowsCursor: resp.next_cursor ?? null,
      flowsHasMore: resp.has_more ?? false,
    });
  } catch (error) {
    console.error("Failed to load flows:", error);
    set({
      error: error instanceof Error ? error.message : "Failed to load flows",
    });
  } finally {
    set({ flowsLoading: false });
  }
}

export async function loadMoreFlowsImpl(
  set: FlowsSetFn,
  cursor: string | null,
  existing: FlowSummary[]
): Promise<void> {
  try {
    set({ flowsLoading: true });
    const resp = await api.listFlowsPage({
      limit: FLOW_LIST_PAGE_SIZE,
      cursor: cursor ?? undefined,
    });
    const moreFlows = (resp.flows || []).map(toFlowSummary);
    set({
      flows: mergeFlowLists(existing, moreFlows),
      flowsCursor: resp.next_cursor ?? cursor,
      flowsHasMore: resp.has_more ?? false,
    });
  } catch (error) {
    console.error("Failed to load more flows:", error);
    set({
      error: error instanceof Error ? error.message : "Failed to load flows",
    });
  } finally {
    set({ flowsLoading: false });
  }
}
