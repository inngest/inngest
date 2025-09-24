import { realtimeMiddleware } from '@inngest/realtime/middleware';
import { Inngest } from 'inngest';

export const inngest = new Inngest({
  id: 'insights-agent-client',
  middleware: [realtimeMiddleware()],
  eventKey: process.env.INNGEST_AI_EVENT_KEY,
});
