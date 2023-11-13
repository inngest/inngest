import { IconExclamationTriangle } from '@inngest/components/icons/ExclamationTriangle';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import { classNames } from '@inngest/components/utils/classNames';
import { renderOutput } from '@inngest/components/utils/outputRenderer';

interface FuncCardFooterProps {
  functionRun: Pick<FunctionRun, 'output' | 'status'>;
}

export function FuncCardFooter({ functionRun }: FuncCardFooterProps) {
  if (!functionRun || !functionRun.output || !functionRun.status) {
    return null;
  }

  const { message, errorName } = renderOutput({
    content: functionRun.output,
    isSuccess: functionRun.status === 'COMPLETED',
  });

  const status = functionRun.status || 'Unknown';

  let content: JSX.Element | null = null;
  let backgroundColor: string = '';

  if (status === 'FAILED' && message && errorName) {
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
