import { createState, type AgentMessageChunk } from '@inngest/agent-kit';
import type { GetFunctionInput } from 'inngest';
import { v4 as uuidv4 } from 'uuid';

import { inngest } from '../client';
import { createChannel } from '../realtime';
import { createInsightsNetwork } from './agents/network';
import type { InsightsAgentState } from './agents/types';

type PublishOpts = {
  topics: string[];
  channel: string;
  runId: string;
};
const publishApi = async ({ topics, channel, runId }: PublishOpts, data: any) => {
  return await inngest['inngestApi'].publish({ topics, channel, runId }, data);
};

export const runAgentNetwork = inngest.createFunction(
  {
    id: 'run-insights-agent',
    name: 'Insights SQL Agent',
  },
  { event: 'insights-agent/chat.requested' },
  async ({
    event,
    publish,
    step,
    runId,
  }: GetFunctionInput<typeof inngest, 'insights-agent/chat.requested'>) => {
    const { threadId: providedThreadId, userMessage, userId, channelKey, history } = event.data;

    // Validate required userId
    if (!userId) {
      throw new Error('userId is required for agent chat execution');
    }

    // Generate a threadId
    let threadId = providedThreadId;
    if (!threadId) {
      threadId = await step.run('generate-thread-id', async () => {
        return uuidv4();
      });
    }

    // Determine the target channel for publishing (channelKey takes priority)
    const targetChannel = channelKey || userId;

    try {
      const clientState = userMessage.state || {};
      const network = createInsightsNetwork(
        threadId,
        createState<InsightsAgentState>(
          {
            userId,
            ...clientState,
          },
          {
            messages: history,
            threadId,
          }
        )
      );

      // Run the network with streaming enabled
      await network.run(userMessage, {
        streaming: {
          publish: async (chunk: AgentMessageChunk) => {
            await publishApi(
              { topics: ['agent_stream'], channel: `user:${targetChannel}`, runId },
              chunk
            );
            // Bring this back when we expose publish right on the Inngest client to use these types
            // await publish(createChannel(targetChannel).agent_stream(chunk));
          },
        },
      });

      return {
        success: true,
        threadId,
        message: 'Agent network completed successfully',
      };
    } catch (error) {
      // Best-effort error event publish; ignore errors here
      const errorChunk: AgentMessageChunk = {
        event: 'error',
        data: {
          error: error instanceof Error ? error.message : 'An unknown error occurred',
          scope: 'network',
          recoverable: true,
          agentId: 'network',
          threadId, // Include threadId for client filtering
          userId, // Include userId for channel routing
        },
        timestamp: Date.now(),
        sequenceNumber: 0,
        id: 'publish-0:network:error',
      };
      try {
        // Use the same target channel as the main flow
        await publishApi(
          { topics: ['agent_stream'], channel: `user:${targetChannel}`, runId },
          errorChunk
        );
        // Bring this back when we expose publish right on the Inngest client to use these types
        // await publish(createChannel(targetChannel).agent_stream(errorChunk));
      } catch {}

      throw error;
    }
  }
);
