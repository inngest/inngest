import { Inngest } from 'inngest';

export const inngest = new Inngest({
  id: 'insights-agent-client',
  eventKey: process.env.INNGEST_AI_EVENT_KEY,
  checkpointing: {
    maxRuntime: 120_000,
  },
});
