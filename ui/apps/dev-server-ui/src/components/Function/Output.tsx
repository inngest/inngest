import { CodeBlock } from '@inngest/components/CodeBlock';
import { renderOutput, type OutputType } from '@inngest/components/utils/outputRenderer';

import { usePrettyJson } from '@/hooks/usePrettyJson';

interface OutputCardProps {
  type: OutputType;
  content: string;
}

export default function OutputCard({ type, content }: OutputCardProps) {
  let { message, errorName, output } = renderOutput({ type, content });

  if (!message && !output) return null;
  let color = 'bg-slate-600';
  if (type === 'completed') {
    color = 'bg-teal-600';
  } else if (type === 'failed') {
    color = 'bg-rose-600/50';
  }

  output = (type === 'completed' && usePrettyJson(output)) || output;

  return (
    <CodeBlock
      header={{ title: errorName, description: message, color: color }}
      tabs={[
        {
          label: type === 'failed' ? 'Stack Trace' : 'Output',
          content: output,
        },
      ]}
    />
  );
}
