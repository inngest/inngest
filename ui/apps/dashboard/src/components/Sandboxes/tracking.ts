import { analytics } from '@/utils/segment';

/**
 * Events tracked via Segment should always follow these patterns:
 * - Name: <Object> <Action, past tense>, using title case, with spaces, ex. "Query Created" "Dashboard Chart Added"
 * - Properties: Always snake_case
 *
 * These events power the Sandboxes fake-door funnel: Viewed -> Waitlist Joined
 * (Join button click) -> Waitlist Submitted (modal Send). Generic Segment
 * pageviews are already captured automatically in SegmentAnalytics.tsx; these
 * add feature-tagged context for a clean funnel.
 */

type SandboxEventName =
  | 'Sandbox Viewed'
  | 'Sandbox Waitlist Joined'
  | 'Sandbox Waitlist Submitted';

type SandboxEventProperties = Record<
  string,
  boolean | number | string | null | undefined
>;

function trackSandboxesEvent(
  event: SandboxEventName,
  properties: SandboxEventProperties = {},
) {
  const compactProperties = Object.fromEntries(
    Object.entries({
      feature: 'sandboxes',
      ...properties,
    }).filter(([, value]) => value !== undefined),
  );

  analytics.track(event, compactProperties);
}

export function trackSandboxesViewed() {
  trackSandboxesEvent('Sandbox Viewed');
}

export function trackSandboxWaitlistJoined() {
  trackSandboxesEvent('Sandbox Waitlist Joined');
}

export function trackSandboxWaitlistSubmitted({
  canContact,
  hasWorkflowDetails,
}: {
  canContact: boolean;
  hasWorkflowDetails: boolean;
}) {
  trackSandboxesEvent('Sandbox Waitlist Submitted', {
    can_contact: canContact,
    has_workflow_details: hasWorkflowDetails,
  });
}
