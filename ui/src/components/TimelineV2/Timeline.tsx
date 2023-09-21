import type { HistoryNode } from './historyParser/historyParser';
import { TimelineNode } from './TimelineNode';

type Props = {
  history: Record<string, HistoryNode>;
};

export function Timeline({ history }: Props) {
  const nodes = Object.values(history).sort(sortAscending);

  let content: JSX.Element | JSX.Element[];
  if (nodes.length === 0) {
    content = <div>No history yet</div>;
  } else {
    content = nodes.map((node) => {
      return <TimelineNode node={node} key={node.groupID} />;
    });
  }

  return <div className="bg-white p-4">{content}</div>;
}

function sortAscending(a: HistoryNode, b: HistoryNode) {
  if (a.startedAt && b.startedAt) {
    return a.startedAt.getTime() - b.startedAt.getTime();
  } else {
    return 0;
  }
}
