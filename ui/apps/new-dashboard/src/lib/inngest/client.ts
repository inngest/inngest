import { realtimeMiddleware } from "@inngest/realtime/middleware";
import type { ChatRequestEvent } from "@inngest/use-agent";
import { EventSchemas, Inngest } from "inngest";

type Events = {
  "insights-agent/chat.requested": {
    data: ChatRequestEvent;
  };
};

export const inngest = new Inngest({
  id: "insights-agent-client",
  middleware: [realtimeMiddleware()],
  eventKey: import.meta.env.INNGEST_AI_EVENT_KEY,
  schemas: new EventSchemas().fromRecord<Events>(),
});
