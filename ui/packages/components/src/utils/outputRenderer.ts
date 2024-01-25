import { maxRenderedOutputSizeBytes } from '@inngest/components/constants';

type RenderedData = {
  message?: string;
  errorName?: string;
  output: string;
};

export function renderOutput({
  isSuccess,
  content,
}: {
  isSuccess: boolean;
  content: string;
}): RenderedData {
  let message = '';
  let errorName = '';
  let output = '';

  if (content) {
    const isOutputTooLarge = content.length > maxRenderedOutputSizeBytes;

    if (!isSuccess && !isOutputTooLarge) {
      try {
        const jsonObject = JSON.parse(content);
        const error = jsonObject?.error ?? jsonObject;
        errorName = error?.name;
        message = error?.message;
        output = error?.stack;
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
