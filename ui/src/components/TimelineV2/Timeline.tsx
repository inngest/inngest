import type { HistoryNode } from './historyParser/historyParser';
import { TimelineNode } from './TimelineNode';

type Props = {
  history: Record<string, HistoryNode>;
};

export function Timeline({ history }: Props) {
  const items = Object.values(history).sort(sortAscending);

  return (
    <div className="bg-white p-4">
      {items.map((item) => {
        return <TimelineNode item={item} key={item.groupID} />;
      })}
    </div>
  );
}

function sortAscending(a: HistoryNode, b: HistoryNode) {
  if (a.startedAt && b.startedAt) {
    return a.startedAt.getTime() - b.startedAt.getTime();
  } else {
    return 0;
  }
}
