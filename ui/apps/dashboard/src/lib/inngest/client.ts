import { realtimeMiddleware } from '@inngest/realtime/middleware';
import type { ChatRequestEvent } from '@inngest/use-agent';
import { EventSchemas, Inngest } from 'inngest';

type Events = {
  'insights-agent/chat.requested': {
    data: ChatRequestEvent;
  };
  'app/csp-violation.reported': {
    data: {};
  };
};

export const inngest = new Inngest({
  id: 'insights-agent-client',
  middleware: [realtimeMiddleware()],
  eventKey: process.env.INNGEST_AI_EVENT_KEY,
  schemas: new EventSchemas().fromRecord<Events>(),
  checkpointing: {
    maxRuntime: 120_000,
  },
});
