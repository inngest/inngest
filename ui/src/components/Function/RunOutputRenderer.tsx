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
        message = jsonObject?.message;
        output = jsonObject?.stack;
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

  console.log(message)

  return {
    message,
    errorName,
    output,
  };
}
