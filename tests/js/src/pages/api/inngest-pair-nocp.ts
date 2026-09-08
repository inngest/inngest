// The fixture shapes with checkpointing OFF. See `inngest-pair-cp`.
import { serve } from "inngest-v4/next";

import { noCpClient, noCpFns } from "@/inngest/canvas_pairs_v4";

export default serve({ client: noCpClient, functions: noCpFns });
