import { FunctionRunStatus } from '@/store/generated';
import { maxRenderedOutputSizeBytes } from '@/utils/constants';
import { FunctionRunExtraStatus, renderRunStatus } from './RunStatus';

type RenderedData = {
  eventName?: string;
  stepName?: string;
  message?: string;
  errorName?: string;
  time?: string;
  status: FunctionRunStatus | FunctionRunExtraStatus;
};

export default function renderFuncCardFooter(functionRun): RenderedData {
  const status = renderRunStatus(functionRun);
  // To do: return event that cancelled the run
  const eventName = functionRun.waitingFor?.eventName || undefined;
  // To do: return step that is running
  const stepName = '';

  const time = new Date(functionRun.waitingFor?.expiryTime).toLocaleTimeString();

  let message = '';
  let errorName = '';

  const isOutputTooLarge = functionRun.output?.length > maxRenderedOutputSizeBytes;
  if (functionRun.output && !isOutputTooLarge) {
    const parsedOutput = JSON.parse(functionRun.output);

    if (parsedOutput.body && typeof parsedOutput.body === 'object') {
      message = parsedOutput.body?.message;
      errorName = parsedOutput.body?.name;
    } else if (parsedOutput.body && typeof parsedOutput.body === 'string') {
      const parsedBody = JSON.parse(parsedOutput.body);
      message = parsedBody?.message;
      errorName = parsedBody?.name;
    }
  }

  return {
    eventName,
    stepName,
    message,
    errorName,
    time,
    status,
  };
}
