import { FunctionRunStatus } from '@/store/generated';
import { maxRenderedOutputSizeBytes } from '@/utils/constants';

type RenderedData = {
  message?: string;
  errorName?: string;
  output: string;
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
        const cleanedJsonString = functionRun.output.slice(1, -1).replace(/\\\"/g, '"');
        try {
          parsedOutput = JSON.parse(cleanedJsonString);
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
  };
}
