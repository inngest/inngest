import { IconStepStatusUnknown } from '@/icons/IconStepStatusUnknown';
import { IconStatusCircleArrowPath } from '@/icons/StatusCircleArrowPath';
import { IconStatusCircleCheck } from '@/icons/StatusCircleCheck';
import { IconStatusCircleCross } from '@/icons/StatusCircleCross';
import { IconStatusCircleMinus } from '@/icons/StatusCircleMinus';
import { IconStatusCircleMoon } from '@/icons/StatusCircleMoon';
import type { HistoryNode } from '../historyParser';

type Props = {
  node: HistoryNode;
};

export function Icon({ node }: Props) {
  let Icon: (props: { className?: string }) => JSX.Element;
  if (node.status === 'cancelled') {
    Icon = IconStatusCircleMinus;
  } else if (node.status === 'completed') {
    Icon = IconStatusCircleCheck;
  } else if (node.status === 'errored') {
      // TODO: Use a different icon to disambiguate an errored and failed steps.
    Icon = IconStatusCircleCross;
  } else if (node.status === 'failed') {
    Icon = IconStatusCircleCross;
  } else if (node.status === 'scheduled') {
    Icon = IconStatusCircleArrowPath;
  } else if (node.status === 'sleeping') {
    Icon = IconStatusCircleMoon;
  } else if (node.status === 'started') {
    if (node.attempt === 0) {
      Icon = IconStatusCircleArrowPath;
    } else {
      // TODO: Use a different icon to disambiguate a retry from a normal step.
      Icon = IconStatusCircleArrowPath;
    }
  } else if (node.status === 'waiting') {
    Icon = IconStatusCircleMoon;
  } else {
    Icon = IconStepStatusUnknown;
  }

  return <Icon className="mr-1 shrink-0" />;
}
