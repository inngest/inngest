import { serve } from 'inngest/next';

import { inngest } from '@/lib/inngest/client';
import { runAgentNetwork } from '@/lib/inngest/functions/run-network';

export const { GET, POST, PUT } = serve({
  client: inngest,
  functions: [runAgentNetwork],
});
