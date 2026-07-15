import { analytics } from '@/utils/segment';

/**
 * Events tracked via Segment should always follow these patterns:
 * - Name: <Object> <Action, past tense>, using title case, with spaces, ex. "Query Created" "Dashboard Chart Added"
 * - Properties: Always snake_case
 */

type SessionEventName =
  | 'Session Empty State Viewed'
  | 'Session Empty State Prompt Copied'
  | 'Session Empty State Example Copied'
  | 'Session Empty State Docs Link Opened';

type SessionEventProperties = Record<
  string,
  boolean | number | string | null | undefined
>;

function trackSessionsEvent(
  event: SessionEventName,
  properties: SessionEventProperties = {},
) {
  const compactProperties = Object.fromEntries(
    Object.entries({
      feature: 'sessions',
      ...properties,
    }).filter(([, value]) => value !== undefined),
  );

  analytics.track(event, compactProperties);
}

export function trackSessionEmptyStateViewed() {
  trackSessionsEvent('Session Empty State Viewed');
}

export function trackSessionEmptyStatePromptCopied() {
  trackSessionsEvent('Session Empty State Prompt Copied');
}

export function trackSessionEmptyStateExampleCopied() {
  trackSessionsEvent('Session Empty State Example Copied');
}

export function trackSessionEmptyStateDocsLinkOpened() {
  trackSessionsEvent('Session Empty State Docs Link Opened');
}
