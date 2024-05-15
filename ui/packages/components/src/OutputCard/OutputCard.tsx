import { CodeBlock } from '@inngest/components/CodeBlock';
import { usePrettyJson } from '@inngest/components/hooks/usePrettyJson';
import { renderOutput } from '@inngest/components/utils/outputRenderer';

interface OutputCardProps {
  isSuccess: boolean;
  content: string;
}

export function OutputCard({ isSuccess, content }: OutputCardProps) {
  let { message, errorName, output } = renderOutput({ isSuccess, content });

  if (!message && !output) return null;
  let color = 'bg-slate-600';
  if (isSuccess) {
    color = 'bg-teal-600';
  } else {
    color = 'bg-rose-600/50';
  }

  output = (isSuccess && usePrettyJson(output)) || output;

  return (
    <CodeBlock
      header={{ title: errorName, description: message, color: color }}
      tabs={[
        {
          label: isSuccess ? 'Output' : 'Stack Trace',
          content: output,
        },
      ]}
    />
  );
}
