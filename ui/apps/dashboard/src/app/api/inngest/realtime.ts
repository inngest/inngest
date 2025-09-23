import { type AgentMessageChunk } from '@inngest/agent-kit';
import { channel, topic } from '@inngest/realtime';

export const createChannel = channel((userId: string) => `user:${userId}`).addTopic(
  topic('agent_stream').type<AgentMessageChunk>()
);
