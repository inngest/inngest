import { IconStatusCancelled } from '@inngest/components/icons/status/Cancelled';
import { IconStatusCompleted } from '@inngest/components/icons/status/Completed';
import { IconStatusErrored } from '@inngest/components/icons/status/Errored';
import { IconStatusFailed } from '@inngest/components/icons/status/Failed';
import { IconStatusRunning } from '@inngest/components/icons/status/Running';
import { IconStatusSleeping } from '@inngest/components/icons/status/Sleeping';
import type { HistoryNode } from '@inngest/components/utils/historyParser';

import type { Timeline } from '..';

export type RenderedData = {
  icon: JSX.Element;
  name: string;
  metadata?: { label: string; value: string };
  badge?: string;
  runLink?: Parameters<React.ComponentProps<typeof Timeline>['navigateToRun']>[0];
};

function getIconForStatus(node: HistoryNode) {
  let icon: JSX.Element;
  if (node.scope === 'function' && node.status === 'started') {
    icon = <IconStatusCompleted />;
  } else if (node.status === 'cancelled') {
    icon = <IconStatusCancelled />;
  } else if (node.status === 'completed') {
    icon = <IconStatusCompleted />;
  } else if (node.status === 'errored') {
    icon = <IconStatusErrored />;
  } else if (node.status === 'failed') {
    icon = <IconStatusFailed />;
  } else if (node.status === 'scheduled' || node.status === 'started') {
    icon = <IconStatusRunning />;
  } else if (node.status === 'sleeping' || node.status === 'waiting') {
    icon = <IconStatusSleeping />;
  } else {
    // TODO: Use a question mark icon or something.
    throw new Error(`unexpected status: ${node.status}`);
  }
  return icon;
}

function getIconsForAttempts({
  attempts,
  icon,
}: {
  attempts: Record<number, HistoryNode>;
  icon: JSX.Element;
}) {
  const attemptsArray = Object.values(attempts);
  const firstAttempt =
    Array.isArray(attemptsArray) && attemptsArray.length > 0 ? attemptsArray[0] : undefined;

  return (
    <span className="flex items-center">
      {firstAttempt && <span className="z-0 -ml-1">{getIconForStatus(firstAttempt)}</span>}
      <span className="bg-canvasBase z-10 -ml-[1.2rem] h-[1.2rem] w-[1.2rem] rounded-full" />
      <span className="z-20 -ml-5">{icon}</span>
    </span>
  );
}

export function renderTimelineNode({
  node,
  isAttempt,
}: {
  node: HistoryNode;
  isAttempt?: boolean;
}): RenderedData {
  const hasRetries = node.attempts && Object.values(node.attempts)?.length > 0;
  let icon: JSX.Element;
  icon = getIconForStatus(node);
  if (hasRetries) {
    icon = getIconsForAttempts({ attempts: node.attempts, icon });
  }

  let name = '...';
  let runLink: RenderedData['runLink'];
  if (node.scope === 'function') {
    name = `Function ${node.status}`;
  } else if (node.scope === 'step') {
    if (isAttempt) {
      name = `Attempt ${node.attempt}`;
    } else if (node.waitForEventConfig) {
      name = node.waitForEventConfig.eventName;
    } else if (node.invokeFunctionConfig) {
      name = node.invokeFunctionConfig.functionID;
    } else if (node.name) {
      name = node.name;
    } else if (node.status === 'scheduled') {
      name = 'Waiting to start next step...';
    } else if (node.status === 'started' && hasRetries) {
      name = 'Running next attempt...';
    } else if (node.status === 'started') {
      name = 'Running next step...';
    }

    if (
      node.invokeFunctionConfig?.functionID &&
      node.invokeFunctionConfig?.eventID &&
      node.invokeFunctionResult?.runID
    ) {
      runLink = {
        eventID: node.invokeFunctionConfig.eventID,
        runID: node.invokeFunctionResult.runID,
        fnID: node.invokeFunctionConfig.functionID,
      };
    }
  }

  let metadata;
  if (node.status === 'cancelled' && node.endedAt) {
    metadata = {
      label: 'Cancelled At:',
      value: node.endedAt.toLocaleString(),
    };
  } else if (node.status === 'completed' && node.endedAt) {
    metadata = {
      label: node.waitForEventResult?.timeout ? 'Timed Out At:' : 'Completed At:',
      value: node.endedAt.toLocaleString(),
    };
  } else if (node.status === 'errored' && !isAttempt) {
    metadata = {
      label: 'Enqueued Retry:',
      value: `${node.attempt + 1}`,
    };
  } else if (node.status === 'errored' && isAttempt && node.endedAt) {
    metadata = {
      label: 'Errored At:',
      value: node.endedAt.toLocaleString(),
    };
  } else if (node.status === 'failed' && node.endedAt) {
    metadata = {
      label: 'Failed At:',
      value: node.endedAt.toLocaleString(),
    };
  } else if (node.status === 'scheduled' && node.scheduledAt) {
    metadata = {
      label: 'Queued At:',
      value: node.scheduledAt.toLocaleString(),
    };
  } else if (node.status === 'sleeping' && node.sleepConfig) {
    metadata = {
      label: 'Sleeping Until:',
      value: node.sleepConfig.until.toLocaleString(),
    };
  } else if (node.status === 'started' && node.startedAt) {
    metadata = {
      label: 'Started At:',
      value: node.startedAt.toLocaleString(),
    };
  } else if (node.status === 'waiting' && node.waitForEventConfig) {
    metadata = {
      label: 'Waiting For:',
      value: node.waitForEventConfig.eventName,
    };
  } else if (node.status === 'waiting' && node.invokeFunctionConfig) {
    metadata = {
      label: 'Waiting For Function:',
      value: node.invokeFunctionConfig.functionID,
    };
  }

  let badge: string | undefined;
  if (node.sleepConfig) {
    badge = 'Sleep';
  } else if (node.waitForEventConfig) {
    badge = 'Wait';
  } else if (node.invokeFunctionConfig) {
    badge = 'Invoke';
  } else if (node.status === 'errored') {
    badge = 'Retry';
  }

  return {
    icon,
    name,
    metadata,
    badge,
    runLink,
  };
}
