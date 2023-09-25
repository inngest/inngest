import TimelineItemHeader from '@/components/AccordionTimeline/TimelineItemHeader';
import { type HistoryNode } from '../historyParser/index';
import { renderMetadata } from './Metadata';
import { renderIcon } from './Icon';
import { renderName } from './Name';

type Props = {
  node: HistoryNode;
};

// type StepKind = 'sleep' | 'waitForEvent';

export function TimelineNode({ node }: Props) {
  // let durationMS: number | undefined = undefined;
  // if (node.scope === 'step' && node.startedAt && node.endedAt) {
  //   durationMS = node.endedAt.getTime() - node.startedAt.getTime();
  // }

  let stepKind: string | undefined;
  if (node.scope === 'function') {
    stepKind = 'Function';
  } else if (node.sleepConfig) {
    stepKind = 'Sleep';
  } else if (node.waitForEventConfig) {
    stepKind = 'Wait For Event';
  }
  const name = renderName({ node: node });
  const metadata = renderMetadata({ node: node });
  const icon = renderIcon({status: node.status})

  return (
    <TimelineItemHeader
      icon={icon}
      badge={stepKind}
      title={name}
      metadata={metadata}
    />
  );
}
