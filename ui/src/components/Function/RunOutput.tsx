import CodeBlock from '@/components/Code/CodeBlock';
import renderRunOutput from '@/components/Function/RunOutputRenderer';
import { usePrettyJson } from '@/hooks/usePrettyJson';
import { FunctionRunStatus } from '@/store/generated';

interface RunOutputCardProps {
  status: FunctionRunStatus;
  content: string;
}

export default function RunOutputCard({ status, content }: RunOutputCardProps) {
  let { message, errorName, output } = renderRunOutput({ status, content });

  if (!message && !output) return null;
  let color = 'bg-slate-600';
  if (status === FunctionRunStatus.Completed) {
    color = 'bg-teal-600';
  } else if (status === FunctionRunStatus.Failed) {
    color = 'bg-rose-600/50';
  }

  output = (status === FunctionRunStatus.Completed && usePrettyJson(output)) || output;

  return (
    <CodeBlock
      header={{ title: errorName, description: message, color: color }}
      tabs={[
        {
          label: status === FunctionRunStatus.Failed ? 'Stack Trace' : 'Output',
          content: output,
        },
      ]}
    />
  );
}
