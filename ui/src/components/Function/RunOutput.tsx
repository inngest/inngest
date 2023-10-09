import CodeBlock from '@/components/Code/CodeBlock';
import renderRunOutput from '@/components/Function/RunOutputRenderer';
import { usePrettyJson } from '@/hooks/usePrettyJson';
import { FunctionRunStatus } from '@/store/generated';
import { type HistoryNode } from '../TimelineV2/historyParser';

interface RunOutputCardProps {
  status: FunctionRunStatus | HistoryNode['status'];
  content: string;
}

export default function RunOutputCard({ status, content }: RunOutputCardProps) {
  let { message, errorName, output } = renderRunOutput({ status, content });

  if (!message && !output) return null;
  let color = 'bg-slate-600';
  if (status === FunctionRunStatus.Completed || status === 'completed') {
    color = 'bg-teal-600';
  } else if (status === FunctionRunStatus.Failed || status === 'failed') {
    color = 'bg-rose-600/50';
  }

  output =
    ((status === FunctionRunStatus.Completed || status === 'completed') && usePrettyJson(output)) ||
    output;

  return (
    <CodeBlock
      header={{ title: errorName, description: message, color: color }}
      tabs={[
        {
          label:
            status === FunctionRunStatus.Failed || status === 'failed' ? 'Stack Trace' : 'Output',
          content: output,
        },
      ]}
    />
  );
}
