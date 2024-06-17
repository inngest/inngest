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

  output = (isSuccess && usePrettyJson(output)) || output;

  return (
    <CodeBlock.Wrapper>
      <CodeBlock
        header={{
          title: isSuccess ? 'Output' : errorName + ': ' + message,
          status: isSuccess ? 'success' : 'error',
        }}
        tab={{
          content: output,
        }}
      />
    </CodeBlock.Wrapper>
  );
}
