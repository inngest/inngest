import { FunctionRunStatus, type FunctionRun } from '@/store/generated';
import { maxRenderedOutputSizeBytes } from '@/utils/constants';

type RenderedData = {
  message?: string;
  errorName?: string;
  output: string;
};

export default function renderRunOutput(
  functionRun: Pick<FunctionRun, 'output' | 'status'>,
): RenderedData {
  let message = '';
  let errorName = '';
  let output = '';

  if (functionRun.output) {
    const isOutputTooLarge = functionRun.output.length > maxRenderedOutputSizeBytes;

    if (functionRun.status === FunctionRunStatus.Failed && !isOutputTooLarge) {
      try {
        const jsonObject = JSON.parse(functionRun.output);
        errorName = jsonObject?.name;
        try {
          const messageObject = JSON.parse(jsonObject.message);
          message = messageObject?.message;
          output = messageObject?.stack;
        } catch (error) {
          console.error("Error parsing 'messageObject' JSON:", error);
        }
      } catch (error) {
        console.error("Error parsing 'jsonObject' JSON:", error);
      }
    } else if (!isOutputTooLarge) {
      if (typeof functionRun.output === 'string') {
        output = functionRun.output;
      } else {
        output = JSON.stringify(functionRun.output, null, 2);
      }
    }
  }

  return {
    message,
    errorName,
    output,
  };
}
