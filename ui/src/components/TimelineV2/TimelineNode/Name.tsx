import type { HistoryNode } from '../historyParser';
import { isEndStatus } from '../historyParser/types';

type Props = {
  node: HistoryNode;
};

export function Name({ node }: Props) {
  if (node.waitForEventConfig) {
    return <>{node.waitForEventConfig.eventName}</>;
  }

  if (node.scope === 'function') {
    return <>Function {node.status}</>;
  }

  if (node.name) {
    return <>{node.name}</>;
  }

  if (node.status === 'errored') {
    return <span className="opacity-50">Errored</span>;
  }

  if (node.status === 'failed' && node.scope === 'step') {
    return <span className="opacity-50">Failed</span>;
  }

  if (node.status === 'scheduled') {
    return <span className="opacity-50">Waiting to start next step...</span>;
  }

  if (node.status === 'started') {
    if (node.attempt === 0) {
      return <span className="opacity-50">Running next step...</span>;
    }

    return <span className="opacity-50">Retrying...</span>;
  }

  return null;
}
