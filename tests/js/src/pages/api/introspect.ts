import { testSdkFunctions } from "@/inngest/sdk_function";
import { testSdkSteps } from "@/inngest/sdk_step_test";
import { testCancel } from "@/inngest/sdk_cancel_test";
import { testRetry } from "@/inngest/sdk_retry_test";
import { testNonRetriableError } from "@/inngest/non_retryable";
import { testParallelism } from "@/inngest/sdk_parallel_test";
import { testWaitForEvent } from "@/inngest/sdk_wait_for_event_test";

export default function(_req: any, res: any) {

  const result: any = [];
  [
    testSdkFunctions,
    testSdkSteps,
    testCancel,
    testRetry,
    testNonRetriableError,
    testParallelism,
    testWaitForEvent,
  ].forEach(f => {
    const data = f["getConfig"](new URL("http://127.0.0.1:3000/api/inngest"), "test-suite");
    result.push(data[0]);
  })

  res.status(200).json(result);
}
