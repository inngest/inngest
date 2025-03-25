import { maxRenderedOutputSizeBytes } from '@inngest/components/constants';
import z from 'zod';

type RenderedData = {
  message?: string;
  errorName?: string;
  output: string;
};

const errorSchema = z.object({
  message: z.string(),
  name: z.string(),
  stack: z.string().optional(),
});

// Handles the old, unwrapped error data and the new, wrapped error data. We'll
// need to handle the old schema for a very long time since it's in TS SDK
// versions <3.12.0
const gracefulErrorSchema = z.union([
  // Old schema
  errorSchema,

  // New schema
  z
    .object({
      error: errorSchema,
    })
    .transform((value) => value.error),
]);

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
        const error = gracefulErrorSchema.parse(JSON.parse(content));
        message = error.message;
        errorName = error.name;
        output = error.stack ?? '';
      } catch (error) {
        console.error("Error parsing 'jsonObject' JSON:", error);
        message = 'Unable to parse error message';
        errorName = 'Unknown Error';
        output = content;
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
