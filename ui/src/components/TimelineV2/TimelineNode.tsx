import { type HistoryNode } from './historyParser/historyParser';

type Props = {
  node: HistoryNode;
};

export function TimelineNode({ node }: Props) {
  let durationMS: number | undefined = undefined;
  if (node.scope === 'step' && node.startedAt && node.endedAt) {
    durationMS = node.endedAt.getTime() - node.startedAt.getTime();
  }

  let name: string;
  if (node.name) {
    name = node.name;
  } else if (node.scope === 'function') {
    name = `Function ${node.status}`;
  } else if (node.waitForEventConfig) {
    name = `Wait for event: ${node.waitForEventConfig.eventName}`;
  } else {
    name = 'To be determined';
  }

  let stepType: 'sleep' | 'waitForEvent' | undefined;
  if (node.waitForEventConfig) {
    stepType = 'waitForEvent';
  } else if (node.sleepConfig) {
    stepType = 'sleep';
  }

  return (
    <div className="border-slate-800 border">
      <p>Name: {name}</p>
      {stepType && <p>Step type: {stepType}</p>}
      <p>Status: {node.status}</p>
      {node.startedAt && <p>Start time: {node.startedAt.toLocaleString()}</p>}
      {durationMS && <p>Duration (MS): {durationMS}</p>}
    </div>
  );
}
