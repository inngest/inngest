import Badge from '@/components/Badge';
import Button from '@/components/Button/Button';
import { IconChevron } from '@/icons/Chevron';
import classNames from '@/utils/classnames';
import { type HistoryNode } from '../historyParser/index';
import { Detail } from './Detail';
import { Icon } from './Icon';
import { Name } from './Name';

type Props = {
  className?: string;
  node: HistoryNode;
};

export function TimelineNode({ className, node }: Props) {
  let durationMS: number | undefined = undefined;
  if (node.scope === 'step' && node.startedAt && node.endedAt) {
    durationMS = node.endedAt.getTime() - node.startedAt.getTime();
  }

  let stepKind: string | undefined;
  if (node.sleepConfig) {
    stepKind = 'Sleep';
  } else if (node.waitForEventConfig) {
    stepKind = 'Wait For Event';
  }

  return (
    <div className={classNames('flex text-white items-start', className)}>
      <span className="flex">
        <Icon node={node} />

        <span className="items-start flex">
          {stepKind && <Badge className="mr-2 whitespace-nowrap">{stepKind}</Badge>}
          <Name node={node} />
        </span>
      </span>

      <span className="flex-grow" />

      {node.scope === 'step' && (
        <>
          <Detail node={node} />

          <span className="shrink-0">
            <Button appearance="solid" icon={<IconChevron />} kind="primary" />
          </span>
        </>
      )}
    </div>
  );
}
