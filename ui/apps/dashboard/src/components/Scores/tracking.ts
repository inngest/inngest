import { analytics } from '@/utils/segment';

/**
 * Events tracked via Segment should always follow these patterns:
 * - Name: <Object> <Action, past tense>, using title case, with spaces, ex. "Query Created" "Dashboard Chart Added"
 * - Properties: Always snake_case
 */

type ScoreEventName =
  | 'Score Empty State Viewed'
  | 'Score Empty State Prompt Copied'
  | 'Score Empty State Example Copied'
  | 'Score Empty State Docs Link Opened';

type ScoreEventProperties = Record<
  string,
  boolean | number | string | null | undefined
>;

function trackScoresEvent(
  event: ScoreEventName,
  properties: ScoreEventProperties = {},
) {
  const compactProperties = Object.fromEntries(
    Object.entries({
      feature: 'scores',
      ...properties,
    }).filter(([, value]) => value !== undefined),
  );

  analytics.track(event, compactProperties);
}

export function trackScoreEmptyStateViewed() {
  trackScoresEvent('Score Empty State Viewed');
}

export function trackScoreEmptyStatePromptCopied() {
  trackScoresEvent('Score Empty State Prompt Copied');
}

export function trackScoreEmptyStateExampleCopied() {
  trackScoresEvent('Score Empty State Example Copied');
}

export function trackScoreEmptyStateDocsLinkOpened() {
  trackScoresEvent('Score Empty State Docs Link Opened');
}
