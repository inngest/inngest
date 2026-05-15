import { realtime } from 'inngest';
import { z } from 'zod';

export type InsightsRealtimeEvent = {
  event: string;
  data: Record<string, unknown>;
  timestamp: number;
};

const insightsRealtimeEventSchema = z.object({
  event: z.string(),
  data: z.record(z.string(), z.unknown()),
  timestamp: z.number(),
});

export const insightsChannel = realtime.channel({
  name: (userId: string) => `user:${userId}`,
  topics: {
    agent_stream: { schema: insightsRealtimeEventSchema },
  },
});
