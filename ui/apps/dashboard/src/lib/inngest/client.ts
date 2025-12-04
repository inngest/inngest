import { realtimeMiddleware } from '@inngest/realtime/middleware';
import type { ChatRequestEvent } from '@inngest/use-agent';
import { EventSchemas, Inngest } from 'inngest';

type Events = {
  'insights-agent/chat.requested': {
    data: ChatRequestEvent;
  };
};

export const inngest = new Inngest({
  id: 'insights-agent-client',
  // @ts-expect-error - realtimeMiddleware is not typed correctly
  middleware: [realtimeMiddleware()],
  eventKey: process.env.INNGEST_AI_EVENT_KEY,
  schemas: new EventSchemas().fromRecord<Events>(),
});
