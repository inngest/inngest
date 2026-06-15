import Anthropic from '@anthropic-ai/sdk';
import { experiment } from 'inngest';
import { v4 as uuidv4 } from 'uuid';

import { inngest } from '../client';
import { insightsChannel } from '../realtime';
import {
  runAgentLoop,
  type InsightsClientState,
  type QueryDraft,
} from './agent/loop';
import { buildSystemPrompt } from './agent/system';
import { insightsTools } from './agent/tools';

type ChatEventData = {
  threadId?: string;
  userMessage: {
    id: string;
    content: string;
    role: 'user';
    state?: Record<string, unknown>;
    clientTimestamp?: string;
    systemPrompt?: string;
  };
  userId?: string;
  accountId?: string;
  requestId?: string;
  channelKey?: string;
  history?: Array<Record<string, unknown>>;
};

export const runInsightsAgent = inngest.createFunction(
  {
    id: 'run-insights-agent',
    name: 'Insights SQL Agent',
    triggers: [{ event: 'insights-agent/chat.requested' }],
  },
  async ({ event, step, group }) => {
    const {
      threadId: providedThreadId,
      userMessage,
      userId,
      accountId,
      requestId,
      channelKey,
      history,
    } = event.data as ChatEventData;

    if (!userId && (!accountId || !requestId)) {
      throw new Error(
        'userId or accountId and requestId is required for agent chat execution',
      );
    }

    const threadId = await step.run('generate-thread-id', () => {
      return providedThreadId || uuidv4();
    });

    const targetChannel = await step.run('generate-target-channel', () => {
      if (channelKey) return channelKey;
      if (userId) return `user:${userId}`;
      return `acct:${accountId}:${requestId}`;
    });

    const clientState = (userMessage.state || {}) as InsightsClientState;

    const ch = insightsChannel(targetChannel);

    await step.realtime.publish('publish-run-started', ch.agent_stream, {
      event: 'run.started',
      data: { threadId, userId },
      timestamp: Date.now(),
    });

    // Select the loop model once (sticky per thread), preserving the experiment.
    const { result: model } = await group.experiment('insights-agent-model', {
      variants: {
        'claude-sonnet-4-5': () =>
          step.run('select-model-sonnet', () => 'claude-sonnet-4-5'),
        'claude-opus-4-8': () =>
          step.run('select-model-opus', () => 'claude-opus-4-8'),
      },
      select: experiment.bucket(threadId, {
        weights: { 'claude-sonnet-4-5': 50, 'claude-opus-4-8': 50 },
      }),
      withVariant: true,
    });

    const anthropicClient = new Anthropic();

    const historyMessages = (history || [])
      .filter(
        (m): m is { role: 'user' | 'assistant'; content: string } =>
          (m.role === 'user' || m.role === 'assistant') &&
          typeof m.content === 'string',
      )
      .map((m) => ({ role: m.role, content: m.content }));

    const draft: QueryDraft = { selectedEvents: [] };

    const result = await runAgentLoop({
      step,
      client: anthropicClient,
      model,
      maxTokens: 4096,
      system: buildSystemPrompt(clientState),
      messages: [
        ...historyMessages,
        { role: 'user', content: userMessage.content },
      ],
      tools: insightsTools,
      ctx: { clientState },
      draft,
      publish: async (id, event, data) => {
        await step.realtime.publish(id, ch.agent_stream, {
          event,
          data: { ...data, threadId },
          timestamp: Date.now(),
        });
      },
      maxIterations: 12,
    });

    await inngest.score({
      name: 'insights_agent_iterations',
      value: result.iterations,
    });
    await inngest.score({
      name: 'insights_agent_tool_calls',
      value: result.toolCalls,
    });
    await inngest.score({
      name: 'insights_agent_output_tokens',
      value: result.tokensOut,
    });

    await step.realtime.publish('publish-run-completed', ch.agent_stream, {
      event: 'run.completed',
      data: {
        threadId,
        sql: draft.sql ?? '',
        title: draft.title ?? '',
        reasoning: draft.reasoning ?? '',
        summary: result.summary,
        tables: draft.tables ?? [],
        needsClarification: !draft.sql,
        selectedEvents: draft.selectedEvents as unknown as Record<
          string,
          unknown
        >,
      },
      timestamp: Date.now(),
    });

    return {
      success: true,
      threadId,
      sql: draft.sql ?? '',
      title: draft.title ?? '',
      summary: result.summary,
      tables: draft.tables ?? [],
      needsClarification: !draft.sql,
      selectedEvents: draft.selectedEvents,
    };
  },
);
