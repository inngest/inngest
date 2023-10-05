import {
  IconStatusCircleArrowPath,
  IconStatusCircleCheck,
  IconStatusCircleCross,
  IconStatusCircleExclamation,
  IconStatusCircleMoon,
} from '@/icons';
import type { HistoryNode } from '../historyParser';

type RenderedData = {
  icon: JSX.Element;
  name: string;
  metadata?: { label: string; value: string };
  badge?: string;
};

export default function renderTimelineNode(node: HistoryNode): RenderedData {
  let icon: JSX.Element;
  if (node.scope === "function" && node.status === "started") {
    icon = <IconStatusCircleCheck />
  } else if (node.status === 'cancelled') {
    icon = <IconStatusCircleCross className="text-slate-700" />;
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
  if (node.scope === "function") {
    name = `Function ${node.status}`;
  } else if (node.scope === "step") {
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
      label: node.waitForEventResult?.timeout
        ? 'Timed Out At:'
        : 'Completed At:',
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
  if (node.scope === 'function') {
    badge = 'Function';
  } else if (node.sleepConfig) {
    badge = 'Sleep';
  } else if (node.waitForEventConfig) {
    badge = 'Wait For Event';
  }

  return {
    icon,
    name,
    metadata,
    badge,
  };
}
