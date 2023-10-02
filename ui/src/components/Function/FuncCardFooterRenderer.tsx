import { FunctionRunStatus } from '@/store/generated';
import { maxRenderedOutputSizeBytes } from '@/utils/constants';

type RenderedData = {
  message?: string;
  errorName?: string;
  status: FunctionRunStatus;
};

export default function renderFuncCardFooter(functionRun): RenderedData {
  let message = '';
  let errorName = '';

  if (functionRun?.status === FunctionRunStatus.Failed) {
    const isOutputTooLarge = functionRun.output?.length > maxRenderedOutputSizeBytes;
    if (functionRun.output && !isOutputTooLarge) {
      let parsedOutput;
      if (typeof functionRun.output === 'string') {
        try {
          parsedOutput = JSON.parse(functionRun.output);
        } catch (error) {
          console.error(`Error parsing payload: `, error);
          parsedOutput = functionRun.output;
        }
      }
      message = parsedOutput?.message;
      errorName = parsedOutput?.name;
    }
  }

  return {
    message,
    errorName,
    status: functionRun?.status,
  };
}
