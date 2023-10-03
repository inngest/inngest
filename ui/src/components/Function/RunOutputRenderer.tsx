import { FunctionRunStatus } from '@/store/generated';
import { maxRenderedOutputSizeBytes } from '@/utils/constants';

type RenderedData = {
  message?: string;
  errorName?: string;
  output: string;
  status: FunctionRunStatus;
};

export default function renderRunOutput(functionRun): RenderedData {
  let message = '';
  let errorName = '';
  let output = '';

  const isOutputTooLarge = functionRun.output?.length > maxRenderedOutputSizeBytes;
  if (functionRun?.status === FunctionRunStatus.Failed) {
    if (functionRun.output && !isOutputTooLarge) {
      let parsedOutput;
      if (typeof functionRun.output === 'string') {
        try {
          parsedOutput = JSON.parse(functionRun.output);
          message = parsedOutput?.message;
          errorName = parsedOutput?.name;
          output = parsedOutput?.stack;
        } catch (error) {
          console.error(`Error parsing payload: `, error);
          parsedOutput = functionRun.output;
        }
      }
    }
  } else if (!isOutputTooLarge) {
    if (typeof functionRun.output === 'string') {
      output = functionRun.output;
    } else {
      output = JSON.stringify(functionRun.output);
    }
  }

  return {
    message,
    errorName,
    output,
    status: functionRun?.status,
  };
}
