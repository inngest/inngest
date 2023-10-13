import { classNames } from '@inngest/components/utils/classNames';

import { IconExclamationTriangle } from '@/icons';
import { FunctionRunStatus, type FunctionRun } from '../../store/generated';
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

  let content: JSX.Element | null = null;
  let backgroundColor: string = '';

  if (status === FunctionRunStatus.Failed && message && errorName) {
    content = (
      <p className="flex items-center gap-2 font-mono">
        <IconExclamationTriangle className="h-3 w-3 text-rose-400" />
        <span className="font-semibold text-rose-400">{errorName}</span>
        <span className="truncate">{message}</span>
      </p>
    );
    backgroundColor = 'bg-rose-600/10';
  }

  if (!content) return null;

  return (
    <div className={classNames(backgroundColor, 'px-5 py-4 text-xs text-slate-100')}>{content}</div>
  );
}
