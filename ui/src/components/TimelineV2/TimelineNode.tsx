import { type HistoryNode } from './historyParser/historyParser';

type Props = {
  item: HistoryNode;
};

export function TimelineNode({ item }: Props) {
  let durationMS: number | undefined = undefined;
  if (item.scope === 'step' && item.startedAt && item.endedAt) {
    durationMS = item.endedAt.getTime() - item.startedAt.getTime();
  }

  let name: string;
  if (item.name) {
    name = item.name;
  } else if (item.scope === 'function') {
    name = `Function ${item.status}`;
  } else if (item.waitForEventConfig) {
    name = `Wait for event: ${item.waitForEventConfig.eventName}`;
  } else {
    name = 'To be determined';
  }

  let stepType: 'sleep' | 'waitForEvent' | undefined;
  if (item.waitForEventConfig) {
    stepType = 'waitForEvent';
  } else if (item.sleepConfig) {
    stepType = 'sleep';
  }

  return (
    <div className="border-slate-800 border">
      <p>Name: {name}</p>
      {stepType && <p>Step type: {stepType}</p>}
      <p>Status: {item.status}</p>
      {item.startedAt && <p>Start time: {item.startedAt.toLocaleString()}</p>}
      {durationMS && <p>Duration (MS): {durationMS}</p>}
    </div>
  );
}
