import { anthropic, experiment } from 'inngest';
import { v4 as uuidv4 } from 'uuid';

import { inngest } from '../client';
import { insightsChannel } from '../realtime';
import {
  buildSystemPrompt as buildEventMatcherPrompt,
  parseToolResult as parseEventMatcherResult,
  selectEventsTool,
} from './agents/event-matcher';
import {
  buildSystemPrompt as buildQueryWriterPrompt,
  parseToolResult as parseQueryWriterResult,
  generateSqlTool,
} from './agents/query-writer';
import {
  buildSystemPrompt as buildSummarizerPrompt,
  parseResult as parseSummarizerResult,
} from './agents/summarizer';

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

    // Extract client state from the user message
    const clientState = (userMessage.state || {}) as {
      eventTypes?: string[];
      schemas?: { name: string; schema: string }[];
      currentQuery?: string;
    };

    const ch = insightsChannel(targetChannel);

    await step.realtime.publish('publish-run-started', ch.agent_stream, {
      event: 'run.started',
      data: { threadId, userId },
      timestamp: Date.now(),
    });

    // ─── Step 1: Event Matcher ─────────────────────────────────────
    const eventMatcherPrompt = await step.run(
      'hydrate-event-matcher-prompt',
      () => {
        const historyMessages = (history || [])
          .filter(
            (
              m,
            ): m is { role: 'user' | 'assistant'; content: string } & Record<
              string,
              unknown
            > =>
              (m.role === 'user' || m.role === 'assistant') &&
              typeof m.content === 'string',
          )
          .map((m) => ({
            role: m.role,
            content: m.content,
          }));

        return {
          system: buildEventMatcherPrompt({
            eventTypes: clientState.eventTypes || [],
            currentQuery: clientState.currentQuery,
          }),
          messages: [
            ...historyMessages,
            { role: 'user' as const, content: userMessage.content },
          ],
        };
      },
    );

    const eventMatcherResult = await step.ai.infer('event-matcher', {
      model: anthropic({
        model: 'claude-haiku-4-5',
        defaultParameters: { max_tokens: 4096 },
      }),
      body: {
        system: eventMatcherPrompt.system,
        messages: eventMatcherPrompt.messages,
        tools: [selectEventsTool],
        tool_choice: { type: 'tool' as const, name: 'select_events' },
      },
    });

    const selectedEventsData = await step.run(
      'extract-event-matcher-result',
      () => {
        return parseEventMatcherResult(
          eventMatcherResult,
          clientState.eventTypes?.length || 0,
        );
      },
    );

    // ─── Step 2: Query Writer ──────────────────────────────────────
    const queryWriterPrompt = await step.run(
      'hydrate-query-writer-prompt',
      () => {
        return {
          system: buildQueryWriterPrompt({
            selectedEvents: selectedEventsData.selectedEvents,
            schemas: clientState.schemas || [],
            currentQuery: clientState.currentQuery,
            query: userMessage.content,
          }),
          messages: [{ role: 'user' as const, content: userMessage.content }],
        };
      },
    );

    // Anthropic API pricing per 1M tokens (hardcoded for now).
    const QUERY_WRITER_PRICING = {
      'claude-sonnet-4-5': { inputPerMTok: 3, outputPerMTok: 15 },
      'claude-opus-4-8': { inputPerMTok: 5, outputPerMTok: 25 },
    } as const;

    const queryWriterBody = {
      system: queryWriterPrompt.system,
      messages: queryWriterPrompt.messages,
      tools: [generateSqlTool],
      tool_choice: { type: 'tool' as const, name: 'generate_sql' },
    };

    const { result: queryWriterResult, variant: queryWriterModel } =
      await group.experiment('query-writer-model', {
        variants: {
          'claude-sonnet-4-5': () =>
            step.ai.infer('query-writer', {
              model: anthropic({
                model: 'claude-sonnet-4-5',
                defaultParameters: { max_tokens: 4096 },
              }),
              body: queryWriterBody,
            }),
          'claude-opus-4-8': () =>
            step.ai.infer('query-writer', {
              model: anthropic({
                model: 'claude-opus-4-8',
                defaultParameters: { max_tokens: 4096 },
              }),
              body: queryWriterBody,
            }),
        },
        select: experiment.weighted({
          'claude-sonnet-4-5': 50,
          'claude-opus-4-8': 50,
        }),
        withVariant: true,
      });

    const usage = (
      queryWriterResult as {
        usage?: { input_tokens?: number; output_tokens?: number };
      }
    ).usage;
    const inputTokens = usage?.input_tokens ?? 0;
    const outputTokens = usage?.output_tokens ?? 0;

    await step.score('score-query-writer-tokens', {
      name: 'query_writer_output_tokens',
      value: outputTokens,
    });

    const pricing =
      QUERY_WRITER_PRICING[
        queryWriterModel as keyof typeof QUERY_WRITER_PRICING
      ];
    const rawCostUsd =
      (inputTokens / 1_000_000) * pricing.inputPerMTok +
      (outputTokens / 1_000_000) * pricing.outputPerMTok;
    // Round to micro-dollars to avoid floating-point noise in the score.
    const costUsd = Math.round(rawCostUsd * 1_000_000) / 1_000_000;
    await step.score('score-query-writer-cost', {
      name: 'query_writer_cost_usd',
      value: costUsd,
    });

    const sqlResult = await step.run('extract-query-writer-result', () => {
      return parseQueryWriterResult(queryWriterResult);
    });

    await step.realtime.publish(
      'publish-query-writer-completed',
      ch.agent_stream,
      {
        event: 'step.completed',
        data: {
          step: 'query-writer',
          threadId,
          sql: sqlResult.sql,
          title: sqlResult.title,
          reasoning: sqlResult.reasoning,
        },
        timestamp: Date.now(),
      },
    );

    // ─── Step 3: Summarizer ────────────────────────────────────────
    const summarizerPrompt = await step.run('hydrate-summarizer-prompt', () => {
      return {
        system: buildSummarizerPrompt({
          selectedEvents: selectedEventsData.selectedEvents,
          sql: sqlResult.sql,
          userIntent: userMessage.content,
        }),
        messages: [{ role: 'user' as const, content: userMessage.content }],
      };
    });

    const summarizerResult = await step.ai.infer('summarizer', {
      model: anthropic({
        model: 'claude-haiku-4-5',
        defaultParameters: { max_tokens: 4096 },
      }),
      body: {
        system: summarizerPrompt.system,
        messages: summarizerPrompt.messages,
      },
    });

    const summary = await step.run('extract-summarizer-result', () => {
      return parseSummarizerResult(summarizerResult);
    });

    await step.realtime.publish('publish-run-completed', ch.agent_stream, {
      event: 'run.completed',
      data: {
        threadId,
        sql: sqlResult.sql,
        title: sqlResult.title,
        reasoning: sqlResult.reasoning,
        summary,
        selectedEvents: selectedEventsData.selectedEvents as unknown as Record<
          string,
          unknown
        >,
      },
      timestamp: Date.now(),
    });

    return {
      success: true,
      threadId,
      sql: sqlResult.sql,
      title: sqlResult.title,
      summary,
      selectedEvents: selectedEventsData.selectedEvents,
    };
  },
);
