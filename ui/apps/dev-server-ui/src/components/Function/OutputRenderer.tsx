import { maxRenderedOutputSizeBytes } from '@inngest/components/constants';

type RenderedData = {
  message?: string;
  errorName?: string;
  output: string;
};

export type OutputType = 'failed' | 'completed';

export default function renderOutput({
  type,
  content,
}: {
  type: OutputType;
  content: string;
}): RenderedData {
  let message = '';
  let errorName = '';
  let output = '';

  if (content) {
    const isOutputTooLarge = content.length > maxRenderedOutputSizeBytes;

    if (type === 'failed' && !isOutputTooLarge) {
      try {
        const jsonObject = JSON.parse(content);
        errorName = jsonObject?.name;
        message = jsonObject?.message;
        output = jsonObject?.stack;
      } catch (error) {
        console.error("Error parsing 'jsonObject' JSON:", error);
      }
    } else if (!isOutputTooLarge) {
      if (typeof content === 'string') {
        output = content;
      } else {
        output = JSON.stringify(content, null, 2);
      }
    }
  }

  return {
    message,
    errorName,
    output,
  };
}
