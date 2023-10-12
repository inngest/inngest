import { IconExclamationTriangle } from '@/icons';
import { FunctionRunStatus, type FunctionRun } from '../../store/generated';
import classNames from '../../utils/classnames';
import renderOutput, { type OutputType } from './OutputRenderer';

interface FuncCardFooterProps {
  functionRun: Omit<FunctionRun, 'history' | 'functionID' | 'historyItemOutput'>;
}

export default function FuncCardFooter({ functionRun }: FuncCardFooterProps) {
  let type: OutputType | undefined;
  if (functionRun?.status === FunctionRunStatus.Completed) {
    type = 'completed';
  } else if (functionRun?.status === FunctionRunStatus.Failed) {
    type = 'failed';
  }

  if (!functionRun || !functionRun.output || !functionRun.status || !type) {
    return null;
  }

  const { message, errorName } = renderOutput({
    content: functionRun.output,
    type,
  });

  const status = functionRun.status || 'Unknown';
  const functionRunStatusFooter = {
    [FunctionRunStatus.Failed]: {
      component: () => {
        if (!message && !errorName) return null;
        return (
          <p className="font-mono flex items-center gap-2">
            <IconExclamationTriangle className="h-3 w-3 text-rose-400" />
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
    <div className={classNames(backgroundColor, 'text-xs px-5 py-4 text-slate-100')}>{content}</div>
  );
}
