import { createState, type AgentMessageChunk, type Message } from '@inngest/agent-kit';
import type { ChatRequestEvent } from '@inngest/use-agent';
import { v4 as uuidv4 } from 'uuid';

import { inngest } from '../client';
import { createChannel } from '../realtime';
import type { InsightsAgentState } from './agents/event-matcher';
import { createInsightsNetwork } from './agents/network';

export const runAgentNetwork = inngest.createFunction(
  {
    id: 'run-insights-agent',
    name: 'Insights SQL Agent',
  },
  { event: 'insights-agent/chat.requested' },
  async ({ event, publish, step }) => {
    const {
      threadId: providedThreadId,
      userMessage,
      userId,
      channelKey,
      history,
    } = event.data as ChatRequestEvent;

    // Validate required userId
    if (!userId) {
      throw new Error('userId is required for agent chat execution');
    }

    // Generate a threadId
    const threadId = await step.run('generate-thread-id', async () => {
      return providedThreadId || uuidv4();
    });

    // Determine the target channel for publishing (channelKey takes priority)
    const targetChannel = await step.run('generate-target-channel', async () => {
      return channelKey || userId;
    });

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
            messages: history as Message[] | undefined,
            threadId,
          }
        )
      );

      // Run the network with streaming enabled
      await network.run(userMessage, {
        streaming: {
          publish: async (chunk: AgentMessageChunk) => {
            await publish(createChannel(targetChannel).agent_stream(chunk));
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
        await publish(createChannel(targetChannel).agent_stream(errorChunk));
      } catch {}

      throw error;
    }
  }
);
