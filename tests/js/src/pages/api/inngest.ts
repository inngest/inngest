import { serve } from "inngest/next";
import { inngest } from "@/inngest/client";

import { testSdkFunctions } from "@/inngest/sdk_function";
import { testSdkSteps } from "@/inngest/sdk_step_test";
import { testCancel } from "@/inngest/sdk_cancel_test";

export default serve(inngest, [testSdkFunctions, testSdkSteps, testCancel]);
