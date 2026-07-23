import { pathCreator } from '@/utils/urls';

// renderRunLink links a run_id identifier (ai_most_expensive_runs,
// ai_slow_runs) to its run detail page.
export function renderRunLink(identifier: string, envSlug: string): React.ReactNode {
  return (
    <a
      className="text-link font-mono hover:underline"
      href={pathCreator.runPopout({ envSlug, runID: identifier })}
    >
      {identifier}
    </a>
  );
}

// renderSessionLink links a session identifier (ai_most_expensive_sessions,
// whose `identifier` is a session id scoped to a session key — see
// InsightsMetricItem.sessionKey) to that session's run list.
export function renderSessionLink(
  sessionId: string,
  sessionKey: string,
  envSlug: string,
): React.ReactNode {
  return (
    <a
      className="text-link font-mono hover:underline"
      href={pathCreator.session({ envSlug, sessionKey, sessionId })}
    >
      {sessionId}
    </a>
  );
}

// renderSessionKeyLink links a session_key (ai_most_expensive_sessions) to
// the results page listing every session under that key.
export function renderSessionKeyLink(sessionKey: string, envSlug: string): React.ReactNode {
  return (
    <a
      className="text-link font-mono hover:underline"
      href={pathCreator.sessions({ envSlug, sessionKey })}
    >
      {sessionKey}
    </a>
  );
}
