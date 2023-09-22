import Badge from '@/components/Badge';
import { IconEvent } from '@/icons/Event';
import type { HistoryNode } from '../historyParser';

type Props = {
  node: HistoryNode;
};

export function Detail({ node }: Props) {
  let content: JSX.Element | undefined;
  if (node.status === 'cancelled' && node.endedAt) {
    content = (
      <>
        Cancelled at
        <Badge className="ml-2">{node.endedAt.toLocaleString()}</Badge>
      </>
    );
  } else if (node.status === 'completed' && node.endedAt) {
    let text = 'Completed at';
    if (node.waitForEventResult?.timeout) {
      text = 'Timed out at';
    }

    content = (
      <>
        <span className="opacity-50">{text}</span>
        <Badge className="ml-2">{node.endedAt.toLocaleString()}</Badge>
      </>
    );
  } else if (node.status === 'errored') {
    content = <>Enqueueing retry {node.attempt + 1}</>;
  } else if (node.status === 'failed' && node.endedAt) {
    content = (
      <>
        Failed at
        <Badge className="ml-2">{node.endedAt.toLocaleString()}</Badge>
      </>
    );
  } else if (node.status === 'scheduled' && node.scheduledAt) {
    content = (
      <>
        Queued at
        <Badge className="flex items-center ml-2">{node.scheduledAt.toLocaleString()}</Badge>
      </>
    );
  } else if (node.status === 'sleeping' && node.sleepConfig) {
    content = (
      <>
        Sleeping until
        <Badge className="flex items-center ml-2">{node.sleepConfig.until.toLocaleString()}</Badge>
      </>
    );
  } else if (node.status === 'started' && node.startedAt) {
    let text: string;
    if (node.attempt === 0) {
      text = 'Started at';
    } else {
      text = `Retried at`;
    }

    content = (
      <>
        {text}
        <Badge className="flex items-center ml-2">{node.startedAt.toLocaleString()}</Badge>
      </>
    );
  } else if (node.status === 'waiting' && node.waitForEventConfig) {
    content = (
      <>
        Waiting for
        <Badge className="flex items-center ml-2">
          <IconEvent className="mr-1" />
          {node.waitForEventConfig.eventName}
        </Badge>
      </>
    );
  }

  if (!content) {
    return null;
  }

  return <span className="items-center flex font-light mr-2 whitespace-nowrap">{content}</span>;
}
