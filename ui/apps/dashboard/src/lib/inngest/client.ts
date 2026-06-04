import { Inngest } from 'inngest';
import { scoreMiddleware } from 'inngest/experimental';

export const inngest = new Inngest({
  id: 'insights-agent-client',
  eventKey: process.env.INNGEST_AI_EVENT_KEY,
  middleware: [scoreMiddleware()],
  checkpointing: {
    maxRuntime: 120_000,
  },
});
