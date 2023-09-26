import TimelineItemHeader from '@/components/AccordionTimeline/TimelineItemHeader';
import { type HistoryNode } from '../historyParser/index';
import renderTimelineNode from './TimelineNodeRenderer';

type Props = {
  node: HistoryNode;
};

export function TimelineNode({ node }: Props) {
  const { icon, badge, name, metadata } = renderTimelineNode(node);

  return (
    <TimelineItemHeader
      icon={icon}
      badge={badge}
      title={name}
      metadata={metadata}
    />
  );
}
