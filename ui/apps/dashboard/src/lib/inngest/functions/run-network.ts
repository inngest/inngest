import { createState, type AgentMessageChunk } from '@inngest/agent-kit';
import type { GetFunctionInput } from 'inngest';
import { v4 as uuidv4 } from 'uuid';

import { inngest } from '../client';
import { createChannel } from '../realtime';
import { createInsightsNetwork } from './agents/network';
import type { InsightsAgentState } from './agents/types';

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
  }: GetFunctionInput<typeof inngest, 'insights-agent/chat.requested'>) => {
    const {
      threadId: providedThreadId,
      userMessage,
      userId,
      channelKey,
      history,
    } = event.data;

    // Validate required userId
    if (!userId) {
      throw new Error('userId is required for agent chat execution');
    }

    // Generate a threadId
    const threadId = await step.run('generate-thread-id', async () => {
      return providedThreadId || uuidv4();
    });

    // Determine the target channel for publishing (channelKey takes priority)
    const targetChannel = await step.run(
      'generate-target-channel',
      async () => {
        return channelKey || userId;
      },
    );

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
          },
        ),
      );

      // Run the network with streaming enabled
      // network.run() returns a NetworkRun instance with the mutated state
      const networkRun = await network.run(userMessage, {
        streaming: {
          publish: async (chunk: AgentMessageChunk) => {
            await publish(createChannel(targetChannel).agent_stream(chunk));
          },
        },
      });

      // Capture summarizer output (doesn't use a tool, just returns text)
      const summarizerResult = networkRun.state.results.find(
        (r) => r.agentName === 'Insights Summarizer',
      );
      const summaryOutput = summarizerResult?.output.find(
        (msg) => msg.type === 'text' && msg.role === 'assistant',
      );
      if (
        summaryOutput &&
        'content' in summaryOutput &&
        typeof summaryOutput.content === 'string'
      ) {
        if (!networkRun.state.data.observability) {
          networkRun.state.data.observability = {};
        }
        if (!networkRun.state.data.observability.summarizer) {
          networkRun.state.data.observability.summarizer = {
            promptContext: {
              selectedEventsCount: 0,
              selectedEventNames: [],
              hasSql: false,
            },
          };
        }
        networkRun.state.data.observability.summarizer.output =
          summaryOutput.content;
      }

      // Capture observability data in a separate step
      await step.run('capture-observability-data', async () => {
        return {
          userPrompt: userMessage.content,
          timestamp: new Date().toISOString(),
          agents: networkRun.state.data.observability || {},
        };
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
          error:
            error instanceof Error
              ? error.message
              : 'An unknown error occurred',
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
  },
);
