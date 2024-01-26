import { serve } from "inngest/next";
import { inngest } from "@/inngest/client";

import { testSdkFunctions } from "@/inngest/sdk_function";
import { testSdkSteps } from "@/inngest/sdk_step_test";
import { testCancel } from "@/inngest/sdk_cancel_test";
import { testRetry } from "@/inngest/sdk_retry_test";
import { testNonRetriableError } from "@/inngest/non_retryable";
import { testParallelism } from "@/inngest/sdk_parallel_test";
import { testWaitForEvent } from "@/inngest/sdk_wait_for_event_test";

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
