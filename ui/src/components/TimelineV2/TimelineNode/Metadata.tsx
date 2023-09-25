import type { HistoryNode } from '../historyParser';

type Props = {
  node: HistoryNode;
};

export function renderMetadata({ node }: Props) {
  let metadata = {
    label: '',
    value: '',
  };
  if (node.status === 'cancelled' && node.endedAt) {
    metadata.label = 'Cancelled At:';
    metadata.value = node.endedAt.toLocaleString();
  } else if (node.status === 'completed' && node.endedAt) {
    metadata.label = 'Completed At:';
    if (node.waitForEventResult?.timeout) {
      metadata.label = 'Timed Out At:';
    }
    metadata.value = node.endedAt.toLocaleString();
  } else if (node.status === 'errored') {
    metadata.label = 'Enqueueing Retry:';
    metadata.value = `${node.attempt + 1}`;
  } else if (node.status === 'failed' && node.endedAt) {
    metadata.label = 'Failed At:';
    metadata.value = node.endedAt.toLocaleString();
  } else if (node.status === 'scheduled' && node.scheduledAt) {
    metadata.label = 'Queued At:';
    metadata.value = node.scheduledAt.toLocaleString();
  } else if (node.status === 'sleeping' && node.sleepConfig) {
    metadata.label = 'Sleeping Until:';
    metadata.value = node.sleepConfig.until.toLocaleString();
  } else if (node.status === 'started' && node.startedAt) {
    metadata.label = 'Started At:';
    metadata.value = node.startedAt.toLocaleString();
  } else if (node.status === 'waiting' && node.waitForEventConfig) {
    metadata.label = 'Waiting For:';
    metadata.value = node.waitForEventConfig.eventName;
  }

  if (metadata.value === '' && metadata.label === '') {
    return undefined;
  }

  return metadata;
}
