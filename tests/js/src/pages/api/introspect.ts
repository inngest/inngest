import { testSdkFunctions } from "@/v3/functions/sdk_function";
import { testSdkSteps } from "@/v3/functions/sdk_step_test";
import { testCancel } from "@/v3/functions/sdk_cancel_test";
import { testRetry } from "@/v3/functions/sdk_retry_test";
import { testNonRetriableError } from "@/v3/functions/non_retryable";
import { testParallelism } from "@/v3/functions/sdk_parallel_test";
import { testWaitForEvent } from "@/v3/functions/sdk_wait_for_event_test";

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
    const data = f["getConfig"]({
      baseUrl: new URL("http://127.0.0.1:3000/api/inngest"),
      appPrefix: "test-suite",
    });
    result.push(data[0]);
  })

  res.status(200).json(result);
}
