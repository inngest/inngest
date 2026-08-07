import { realtime } from 'inngest';
import { z } from 'zod';

// Publish-only: the last consumer of this channel is the cloud's
// POST /v2/insights/query/prompt, which subscribes server-side before sending
// its event. The Insights chat UI reads run output instead — see RunWatcher.
const insightsRealtimeEventSchema = z.object({
  event: z.string(),
  data: z.record(z.string(), z.unknown()),
  timestamp: z.number(),
});

export const insightsChannel = realtime.channel({
  name: (targetChannel: string) => targetChannel,
  topics: {
    agent_stream: { schema: insightsRealtimeEventSchema },
  },
});
