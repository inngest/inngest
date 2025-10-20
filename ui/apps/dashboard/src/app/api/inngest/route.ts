import { serve } from 'inngest/next';

import { inngest } from './client';
import { runAgentNetwork } from './functions/run-network';

export const { GET, POST, PUT } = serve({
  client: inngest,
  functions: [runAgentNetwork],
});
