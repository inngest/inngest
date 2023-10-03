import CodeBlock from '@/components/Code/CodeBlock';
import renderRunOutput from '@/components/Function/RunOutputRenderer';
import { FunctionRunStatus, type FunctionRun } from '@/store/generated';

interface RunOutputCardProps {
  functionRun: Omit<FunctionRun, 'history' | 'functionID' | 'historyItemOutput'>;
}

export default function RunOutputCard({ functionRun }: RunOutputCardProps) {
  const { message, errorName, output } = renderRunOutput(functionRun);

  if (!message && !output) return null;
  let color = 'bg-slate-600';
  if (functionRun.status === FunctionRunStatus.Completed) {
    color = 'bg-teal-600';
  } else if (functionRun.status === FunctionRunStatus.Failed) {
    color = 'bg-rose-600/50';
  }

  return (
    <CodeBlock
      header={{ title: errorName, description: message, color: color }}
      tabs={[
        {
          label: functionRun.status === FunctionRunStatus.Failed ? 'Stack Trace' : 'Output',
          content: output,
        },
      ]}
    />
  );
}
