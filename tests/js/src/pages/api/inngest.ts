import { serve } from "inngest/next";
import { inngest } from "@/v3/client";

import { testSdkFunctions } from "@/v3/functions/sdk_function";
import { testSdkSteps } from "@/v3/functions/sdk_step_test";
import { testCancel } from "@/v3/functions/sdk_cancel_test";
import { testRetry } from "@/v3/functions/sdk_retry_test";
import { testNonRetriableError } from "@/v3/functions/non_retryable";
import { testParallelism } from "@/v3/functions/sdk_parallel_test";
import { testWaitForEvent } from "@/v3/functions/sdk_wait_for_event_test";

export default serve({
  client: inngest,
  functions: [
    testSdkFunctions,
    testSdkSteps,
    testCancel,
    testRetry,
    testNonRetriableError,
    testParallelism,
    testWaitForEvent,
  ],
});
