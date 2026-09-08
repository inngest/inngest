// The fixture shapes with checkpointing ON. Its twin at `inngest-pair-nocp`
// serves the same shapes with it off; the two register as separate apps so a
// run can be captured either way.
import { serve } from "inngest-v4/next";

import { cpClient, cpFns } from "@/inngest/canvas_pairs_v4";

export default serve({ client: cpClient, functions: cpFns });
