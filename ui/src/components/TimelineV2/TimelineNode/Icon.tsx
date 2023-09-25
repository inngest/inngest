import { IconStatusCircleArrowPath } from '@/icons/StatusCircleArrowPath';
import { IconStatusCircleCheck } from '@/icons/StatusCircleCheck';
import { IconStatusCircleCross } from '@/icons/StatusCircleCross';
import { IconStatusCircleExclamation } from '@/icons/StatusCircleExclamation';
import { IconStatusCircleMoon } from '@/icons/StatusCircleMoon';
import type { HistoryNode } from '../historyParser';

type Props = {
  status: HistoryNode['status'];
};

export function renderIcon({ status }: Props) {
  let Icon: JSX.Element;
  if (status === 'cancelled') {
    Icon = <IconStatusCircleCross className="text-slate-700" />;
  } else if (status === 'completed') {
    Icon = <IconStatusCircleCheck />;
  } else if (status === 'errored') {
    Icon = <IconStatusCircleExclamation />;
  } else if (status === 'failed') {
    Icon = <IconStatusCircleCross />;
  } else if (status === 'scheduled') {
    Icon = <IconStatusCircleArrowPath />;
  } else if (status === 'sleeping') {
    Icon = <IconStatusCircleMoon />;
  } else if (status === 'started') {
    Icon = <IconStatusCircleArrowPath />;
  } else if (status === 'waiting') {
    Icon = <IconStatusCircleMoon />;
  } else {
    // TODO: Use a question mark icon or something.
    throw new Error(`unexpected status: ${status}`);
  }

  return Icon;
}
