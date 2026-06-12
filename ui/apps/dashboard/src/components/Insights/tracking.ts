import { analytics } from '@/utils/segment';
import { UNTITLED_QUERY } from './InsightsTabManager/constants';
import type { InsightsFetchResult } from './InsightsStateMachineContext/types';
import type { Tab } from './types';

/**
 * Events tracked via Segment should always follow these patterns:
 * - Name: <Object> <Action, past tense>, using title case, with spaces, ex. "Query Created" "Dashboard Chart Added"
 * - Properties: Always snake_case
 */

export type InsightsQueryRunTrigger =
  | 'ai_assistant'
  | 'button'
  | 'context_menu'
  | 'keyboard'
  | 'unknown';

type InsightsQueryRunResult = 'failure' | 'success';

type InsightsEventName =
  | 'Insights AI Message Sent'
  | 'Insights Query Ran'
  | 'Insights Query Saved'
  | 'Insights Query Shared'
  | 'Insights Results Downloaded';

type InsightsEventProperties = Record<
  string,
  boolean | number | string | null | undefined
>;

type QueryMetadataArgs = {
  query: string;
  queryName: string;
  savedQueryId?: string;
  tabId: string;
};

function trackInsightsEvent(
  event: InsightsEventName,
  properties: InsightsEventProperties = {},
) {
  const compactProperties = Object.fromEntries(
    Object.entries({
      feature: 'insights',
      ...properties,
    }).filter(([, value]) => value !== undefined),
  );

  analytics.track(event, compactProperties);
}

function getQueryMetadata({
  query,
  queryName,
  savedQueryId,
  tabId,
}: QueryMetadataArgs): InsightsEventProperties {
  const trimmedQuery = query.trim();

  return {
    is_saved_query: savedQueryId !== undefined,
    query_length: trimmedQuery.length,
    query_line_count:
      trimmedQuery === '' ? 0 : trimmedQuery.split(/\r?\n/).length,
    query_name_set: queryName.trim() !== '' && queryName !== UNTITLED_QUERY,
    saved_query_id: savedQueryId,
    tab_id: tabId,
  };
}

function getTabMetadata(tab: Tab): InsightsEventProperties {
  return getQueryMetadata({
    query: tab.query,
    queryName: tab.name,
    savedQueryId: tab.savedQueryId,
    tabId: tab.id,
  });
}

function getDiagnosticsMetadata(
  data: InsightsFetchResult | undefined,
): InsightsEventProperties {
  if (!data) return {};

  return {
    diagnostic_error_count: data.diagnostics.filter(
      (diagnostic) => diagnostic.severity === 'error',
    ).length,
    diagnostic_info_count: data.diagnostics.filter(
      (diagnostic) => diagnostic.severity === 'info',
    ).length,
    diagnostic_warning_count: data.diagnostics.filter(
      (diagnostic) => diagnostic.severity === 'warning',
    ).length,
  };
}

function getResultMetadata(
  data: InsightsFetchResult | undefined,
): InsightsEventProperties {
  if (!data) return {};

  return {
    column_count: data.columns.length,
    row_count: data.rows.length,
    ...getDiagnosticsMetadata(data),
  };
}

export function trackInsightsQueryRan({
  data,
  durationMs,
  errorType,
  result,
  trigger,
  ...metadata
}: QueryMetadataArgs & {
  data?: InsightsFetchResult;
  durationMs: number;
  errorType?: 'diagnostic' | 'network';
  result: InsightsQueryRunResult;
  trigger: InsightsQueryRunTrigger;
}) {
  trackInsightsEvent('Insights Query Ran', {
    ...getQueryMetadata(metadata),
    ...getResultMetadata(data),
    duration_ms: durationMs,
    error_type: errorType,
    result,
    trigger,
  });
}

export function trackInsightsQuerySaved({
  queryId,
  tab,
}: {
  queryId: string;
  tab: Tab;
}) {
  trackInsightsEvent('Insights Query Saved', {
    ...getTabMetadata(tab),
    is_saved_query: true,
    query_id: queryId,
    saved_query_id: queryId,
  });
}

export function trackInsightsQueryShared({ queryId }: { queryId: string }) {
  trackInsightsEvent('Insights Query Shared', {
    query_id: queryId,
  });
}

export function trackInsightsResultsDownloaded({
  data,
  format,
  queryName,
}: {
  data: InsightsFetchResult;
  format: 'csv' | 'json';
  queryName?: string;
}) {
  trackInsightsEvent('Insights Results Downloaded', {
    ...getResultMetadata(data),
    format,
    query_name_set:
      queryName !== undefined &&
      queryName.trim() !== '' &&
      queryName !== UNTITLED_QUERY,
  });
}

export function trackInsightsAIMessageSent({
  content,
  eventTypeCount,
  hasCurrentQuery,
  historyMessageCount,
  schemaCount,
  threadId,
}: {
  content: string;
  eventTypeCount: number;
  hasCurrentQuery: boolean;
  historyMessageCount: number;
  schemaCount: number;
  threadId: string;
}) {
  trackInsightsEvent('Insights AI Message Sent', {
    event_type_count: eventTypeCount,
    has_current_query: hasCurrentQuery,
    history_message_count: historyMessageCount,
    message_length: content.trim().length,
    schema_count: schemaCount,
    thread_id: threadId,
  });
}
