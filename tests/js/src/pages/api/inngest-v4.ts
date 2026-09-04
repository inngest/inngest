// Separate serve endpoint for the v4 canvas fixtures, so the existing v3 app at
// /api/inngest is untouched and the two SDK versions register as distinct apps.
import { serve } from "inngest-v4/next";

import {
  inngestV4Fns,
  v4Chains,
  v4Parallel,
  v4Sequential,
  v4Slow,
  v4Unbalanced,
  v4Nested,
  v4Race,
  v4MixedOutcomes,
  v4SleepInBranch,
  v4DupeNames,
  v4ForeignAsync,
  v4Dynamic,
  v4Deep3,
  v4Stress,
  v4NestedInBranch,
  v4Pathological,
  v4DeadEnd,
  v4Cancel,
  v4Tall,
  v4Contended,
} from "@/inngest/canvas_shapes_v4";

export default serve({
  client: inngestV4Fns,
  functions: [
    v4Parallel,
    v4Chains,
    v4Sequential,
    v4Slow,
    v4Unbalanced,
    v4Nested,
    v4Race,
    v4MixedOutcomes,
    v4SleepInBranch,
    v4DupeNames,
    v4ForeignAsync,
    v4Dynamic,
    v4Deep3,
    v4Stress,
    v4NestedInBranch,
    v4Pathological,
    v4DeadEnd,
    v4Cancel,
    v4Tall,
    v4Contended,
  ],
});
