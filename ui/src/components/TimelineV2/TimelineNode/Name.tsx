import type { HistoryNode } from '../historyParser';
import { isEndStatus } from '../historyParser/types';

type Props = {
  node: HistoryNode;
};

export function renderName({ node }: Props) {
  let name = '...';
  if (node.waitForEventConfig) {
    name = node.waitForEventConfig.eventName;
  } else if (node.name) {
    name = node.name;
  } else if (node.status === 'scheduled') {
    name = 'Waiting to start next step...';
  } else if (node.status === 'started') {
    name = 'Running next step...';
  }

  return name;
}
