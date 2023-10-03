import { IconExclamationTriangle } from '@/icons';
import { FunctionRunStatus, type FunctionRun } from '../../store/generated';
import classNames from '../../utils/classnames';
import renderRunOutput from './RunOutputRenderer';

interface FuncCardFooterProps {
  functionRun: Omit<FunctionRun, 'history' | 'functionID' | 'historyItemOutput'>;
}

export default function FuncCardFooter({ functionRun }: FuncCardFooterProps) {
  const { message, errorName, status } = renderRunOutput(functionRun);
  const functionRunStatusFooter = {
    [FunctionRunStatus.Failed]: {
      component: () => {
        if (!message && !errorName) return null;
        return (
          <p className="font-mono flex items-center gap-2">
            <IconExclamationTriangle className="icon-2xs text-rose-400" />
            <span className="text-rose-400 font-semibold">{errorName}</span>
            <span className="truncate">{message}</span>
          </p>
        );
      },
      color: 'bg-rose-600/10',
    },
  } as const;

  const content =
    functionRunStatusFooter[status]?.component && functionRunStatusFooter[status].component();
  const backgroundColor =
    functionRunStatusFooter[status]?.color && functionRunStatusFooter[status].color;

  if (!content) return null;

  return (
    <div className={classNames(backgroundColor, 'text-2xs px-5 py-4 text-slate-100')}>
      {content}
    </div>
  );
}
