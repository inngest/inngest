import Anthropic from '@anthropic-ai/sdk';
import { anthropic, experiment } from 'inngest';
import { createScorer } from 'inngest/experimental';
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

// Deferred LLM-as-judge: rates how well the generated SQL fits the user's chat
// context (0 = poor fit, 1 = perfect fit). Runs after the parent run finalizes;
// passing `experiment` on the defer call attributes the score to the selected
// query-writer-model variant.
export const insightsJudgeScorer = createScorer(
  inngest,
  { id: 'insights-judge-relevance' },
  async ({ event, step }) => {
    const { sql, chatContext } = event.data as {
      sql: string;
      chatContext: string;
    };

    const result = await step.ai.infer('judge-relevance', {
      model: anthropic({
        model: 'claude-haiku-4-5',
        defaultParameters: { max_tokens: 1024 },
      }),
      body: {
        system:
          "You evaluate a SQL query an assistant generated against the user's " +
          'chat context. Rate how well the query fits what the user asked for, ' +
          'then call submit_score with a number from 0 (poor fit) to 1 ' +
          '(perfect fit).',
        messages: [
          {
            role: 'user' as const,
            content: `User chat context:\n${chatContext}\n\nGenerated SQL:\n${sql}`,
          },
        ],
        tools: [
          {
            name: 'submit_score' as const,
            description: 'Submit the relevance score for the generated SQL.',
            input_schema: {
              type: 'object' as const,
              properties: {
                relevance: {
                  type: 'number',
                  description:
                    'How well the SQL fits the user request, 0 to 1.',
                },
              },
              required: ['relevance'],
            },
          },
        ],
        tool_choice: { type: 'tool' as const, name: 'submit_score' },
      },
    });

    const toolUse = (
      result as {
        content: Array<{ type: string; name?: string; input?: unknown }>;
      }
    ).content.find(
      (block) => block.type === 'tool_use' && block.name === 'submit_score',
    );
    const relevance = (toolUse?.input as { relevance?: number } | undefined)
      ?.relevance;

    return { name: 'insights_judge_relevance', value: relevance ?? 0 };
  },
);

export const runInsightsAgent = inngest.createFunction(
  {
    id: 'run-insights-agent',
    name: 'Insights SQL Agent',
    triggers: [{ event: 'insights-agent/chat.requested' }],
  },
  async ({ event, step, group, defer }) => {
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

    // Cost in USD from token counts and per-1M-token pricing.
    const calculateCostUsd = (
      inputTokens: number,
      outputTokens: number,
      pricing: { inputPerMTok: number; outputPerMTok: number },
    ) =>
      (inputTokens / 1_000_000) * pricing.inputPerMTok +
      (outputTokens / 1_000_000) * pricing.outputPerMTok;

    const queryWriterBody = {
      system: queryWriterPrompt.system,
      messages: queryWriterPrompt.messages,
      tools: [generateSqlTool],
      tool_choice: { type: 'tool' as const, name: 'generate_sql' },
    };

    const anthropicClient = new Anthropic();

    const { result: queryWriterResult, experimentRef } = await group.experiment(
      'query-writer-model',
      {
        variants: {
          'claude-sonnet-4-5': () =>
            step.run('query-writer', async () => {
              const startedAt = Date.now();
              const result = await anthropicClient.messages.create({
                model: 'claude-sonnet-4-5',
                max_tokens: 4096,
                ...queryWriterBody,
              });
              const latencyMs = Date.now() - startedAt;

              const inputTokens = result.usage.input_tokens;
              const outputTokens = result.usage.output_tokens;
              const pricing = QUERY_WRITER_PRICING['claude-sonnet-4-5'];
              const costUsd = calculateCostUsd(
                inputTokens,
                outputTokens,
                pricing,
              );

              await inngest.score({
                name: 'query_writer_latency_ms',
                value: latencyMs,
              });
              await inngest.score({
                name: 'query_writer_output_tokens',
                value: outputTokens,
              });
              await inngest.score({
                name: 'query_writer_cost_usd',
                value: costUsd,
              });

              return result;
            }),
          'claude-opus-4-8': () =>
            step.run('query-writer', async () => {
              const startedAt = Date.now();
              const result = await anthropicClient.messages.create({
                model: 'claude-opus-4-8',
                max_tokens: 4096,
                ...queryWriterBody,
              });
              const latencyMs = Date.now() - startedAt;

              const inputTokens = result.usage.input_tokens;
              const outputTokens = result.usage.output_tokens;
              const pricing = QUERY_WRITER_PRICING['claude-opus-4-8'];
              const costUsd = calculateCostUsd(
                inputTokens,
                outputTokens,
                pricing,
              );

              await inngest.score({
                name: 'query_writer_latency_ms',
                value: latencyMs,
              });
              await inngest.score({
                name: 'query_writer_output_tokens',
                value: outputTokens,
              });
              await inngest.score({
                name: 'query_writer_cost_usd',
                value: costUsd,
              });

              return result;
            }),
        },
        select: experiment.weighted({
          'claude-sonnet-4-5': 50,
          'claude-opus-4-8': 50,
        }),
      },
    );

    const sqlResult = await step.run('extract-query-writer-result', () => {
      return parseQueryWriterResult(queryWriterResult);
    });

    // Fire-and-forget LLM-as-judge scoring the SQL's fit to the user's chat
    // context, attributed to the selected query-writer-model variant.
    const chatContext = [
      ...(history || [])
        .map((m) => `${String(m.role ?? '')}: ${String(m.content ?? '')}`)
        .filter((line) => line.trim() !== ':'),
      `user: ${userMessage.content}`,
    ].join('\n');
    defer('judge-relevance', {
      function: insightsJudgeScorer,
      experiment: experimentRef,
      data: { sql: sqlResult.sql, chatContext },
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
