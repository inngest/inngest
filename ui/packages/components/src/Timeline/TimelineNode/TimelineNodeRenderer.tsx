import { IconStatusCircleArrowPath } from '@inngest/components/icons/StatusCircleArrowPath';
import { IconStatusCircleCheck } from '@inngest/components/icons/StatusCircleCheck';
import { IconStatusCircleCross } from '@inngest/components/icons/StatusCircleCross';
import { IconStatusCircleExclamation } from '@inngest/components/icons/StatusCircleExclamation';
import { IconStatusCircleMinus } from '@inngest/components/icons/StatusCircleMinus';
import { IconStatusCircleMoon } from '@inngest/components/icons/StatusCircleMoon';
import type { HistoryNode } from '@inngest/components/utils/historyParser';

type RenderedData = {
  icon: JSX.Element;
  name: string;
  metadata?: { label: string; value: string };
  badge?: string;
};

export function renderTimelineNode(node: HistoryNode): RenderedData {
  let icon: JSX.Element;
  if (node.scope === 'function' && node.status === 'started') {
    icon = <IconStatusCircleCheck />;
  } else if (node.status === 'cancelled') {
    icon = <IconStatusCircleMinus />;
  } else if (node.status === 'completed') {
    icon = <IconStatusCircleCheck />;
  } else if (node.status === 'errored') {
    icon = <IconStatusCircleExclamation />;
  } else if (node.status === 'failed') {
    icon = <IconStatusCircleCross />;
  } else if (node.status === 'scheduled' || node.status === 'started') {
    icon = <IconStatusCircleArrowPath />;
  } else if (node.status === 'sleeping' || node.status === 'waiting') {
    icon = <IconStatusCircleMoon />;
  } else {
    // TODO: Use a question mark icon or something.
    throw new Error(`unexpected status: ${node.status}`);
  }

  let name = '...';
  if (node.scope === 'function') {
    name = `Function ${node.status}`;
  } else if (node.scope === 'step') {
    if (node.waitForEventConfig) {
      name = node.waitForEventConfig.eventName;
    } else if (node.name) {
      name = node.name;
    } else if (node.status === 'scheduled') {
      name = 'Waiting to start next step...';
    } else if (node.status === 'started') {
      name = 'Running next step...';
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
  } else if (node.status === 'errored') {
    metadata = {
      label: 'Enqueueing Retry:',
      value: `${node.attempt + 1}`,
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
  }

  let badge: string | undefined;
  if (node.sleepConfig) {
    badge = 'Sleep';
  } else if (node.waitForEventConfig) {
    badge = 'Wait';
  } else if (node.status === 'errored') {
    badge = 'Retry';
  }

  return {
    icon,
    name,
    metadata,
    badge,
  };
}
