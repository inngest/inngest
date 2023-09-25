import type { HistoryNode } from '../historyParser';
import { isEndStatus } from '../historyParser/types';

type Props = {
  node: HistoryNode;
};

export function Name({ node }: Props) {
  if (node.waitForEventConfig) {
    return <>{node.waitForEventConfig.eventName}</>;
  }

  if (node.name) {
    return <>{node.name}</>;
  }

  if (node.status === 'scheduled') {
    return <span className="opacity-50">Waiting to start next step...</span>;
  }

  if (node.status === 'started') {
    return <span className="opacity-50">Running next step...</span>;
  }

  return null;
}
