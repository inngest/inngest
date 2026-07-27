import { experiment } from 'inngest';
import { createScorer } from 'inngest/experimental';
import { v4 as uuidv4 } from 'uuid';

import { inngest } from '../client';
import { insightsChannel } from '../realtime';
import {
  parseToolArguments,
  runAgentLoop,
  type ChatCompletion,
  type InsightsClientState,
  type QueryDraft,
} from './agent/loop';
import { buildSystemPrompt } from './agent/system';
import { insightsTools, validateQueryTool } from './agent/tools';

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
  // True when a browser is subscribed to the agent stream and can execute
  // validate_query round trips (set by /api/chat; absent for the headless API).
  canValidate?: boolean;
};

// All LLM calls route through OpenRouter. Requires OPENROUTER_API_KEY.
const OPENROUTER_CHAT_URL = 'https://openrouter.ai/api/v1/chat/completions';

// Internal model ids, kept stable for pricing and experiment attribution, mapped
// to the OpenRouter slugs the API expects.
const OPENROUTER_MODEL_IDS: Record<string, string> = {
  'claude-opus-5': 'anthropic/claude-opus-5',
  'glm-5.2': 'z-ai/glm-5.2',
  'claude-haiku-4-5': 'anthropic/claude-haiku-4.5',
};

// Throws on a non-2xx response so the enclosing step.run retries.
async function chatCompletion(
  model: string,
  body: Record<string, unknown>,
): Promise<ChatCompletion> {
  const res = await fetch(OPENROUTER_CHAT_URL, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${process.env.OPENROUTER_API_KEY ?? ''}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      ...body,
      // Last so the resolved model always wins over a stray `model` in `body`.
      model: OPENROUTER_MODEL_IDS[model] ?? model,
    }),
  });
  if (!res.ok) {
    throw new Error(
      `OpenRouter request failed (${res.status}): ${await res.text()}`,
    );
  }
  return (await res.json()) as ChatCompletion;
}

// Deferred LLM-as-judge, run after the parent run finalizes. Two modes:
// - SQL produced → insights_judge_relevance: how well the query fits the chat
//   context (0 = poor fit, 1 = perfect fit).
// - no SQL (clarification or general answer) → insights_judge_no_query_appropriate:
//   whether skipping the query was right (1) or the user clearly wanted one (0).
//   Its average is the inverse of the agent's submit-miss rate.
// Passing `experiment` on the defer call attributes the score to the selected
// query-writer-model variant.
export const insightsJudgeScorer = createScorer(
  inngest,
  { id: 'insights-judge-relevance' },
  async ({ event, step }) => {
    const { sql, summary, chatContext } = event.data as {
      sql: string;
      summary: string;
      chatContext: string;
    };

    const system = sql
      ? "You evaluate a SQL query an assistant generated against the user's " +
        'chat context. Rate how well the query fits what the user asked for, ' +
        'then call submit_score with a number from 0 (poor fit) to 1 ' +
        '(perfect fit).'
      : 'An assistant chose to respond WITHOUT generating a SQL query — it ' +
        'asked a clarifying question or answered a general question instead. ' +
        'Given the chat context, call submit_score with 1 if that was ' +
        'appropriate, or 0 if the user clearly asked for a query.';
    const content = sql
      ? `User chat context:\n${chatContext}\n\nGenerated SQL:\n${sql}`
      : `User chat context:\n${chatContext}\n\nAssistant response (no SQL):\n${summary}`;

    const result = (await step.run('judge-relevance', () =>
      chatCompletion('claude-haiku-4-5', {
        max_tokens: 1024,
        messages: [
          { role: 'system', content: system },
          { role: 'user', content },
        ],
        tools: [
          {
            type: 'function',
            function: {
              name: 'submit_score',
              description: 'Submit the relevance score for the generated SQL.',
              parameters: {
                type: 'object',
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
          },
        ],
        tool_choice: { type: 'function', function: { name: 'submit_score' } },
      }),
    )) as ChatCompletion;

    const toolCall = result.choices[0]?.message.tool_calls?.find(
      (c) => c.function.name === 'submit_score',
    );
    const relevance = toolCall
      ? (parseToolArguments(toolCall.function.arguments).relevance as
          | number
          | undefined)
      : undefined;

    return {
      name: sql
        ? 'insights_judge_relevance'
        : 'insights_judge_no_query_appropriate',
      value: relevance ?? 0,
    };
  },
);

export const runInsightsAgent = inngest.createFunction(
  {
    id: 'run-insights-agent',
    name: 'Insights SQL Agent',
    triggers: [{ event: 'insights-agent/chat.requested' }],
    // Runs after all step retries exhaust; without it the chat UI spins
    // forever waiting for a run.completed that will never arrive.
    onFailure: async ({ event, step }) => {
      const original = event.data.event.data as ChatEventData;
      const targetChannel =
        original.channelKey ||
        (original.userId
          ? `user:${original.userId}`
          : `acct:${original.accountId}:${original.requestId}`);
      await step.realtime.publish(
        'publish-run-error',
        insightsChannel(targetChannel).agent_stream,
        {
          event: 'error',
          data: {
            threadId: original.threadId ?? '',
            error:
              'The Insights agent could not complete this request. Please try again.',
          },
          timestamp: Date.now(),
        },
      );
    },
  },
  async ({ event, step, group, defer, runId }) => {
    const {
      threadId: providedThreadId,
      userMessage,
      userId,
      accountId,
      requestId,
      channelKey,
      history,
      canValidate,
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
    const clientState = (userMessage.state || {}) as InsightsClientState;

    const ch = insightsChannel(targetChannel);

    await step.realtime.publish('publish-run-started', ch.agent_stream, {
      event: 'run.started',
      data: { threadId, userId },
      timestamp: Date.now(),
    });

    // Select the model once up front; the loop reuses it every iteration.
    const { result: model, experimentRef } = await group.experiment(
      'query-writer-model',
      {
        variants: {
          'claude-opus-5': () =>
            step.run('select-model', () => 'claude-opus-5'),
          'glm-5.2': () => step.run('select-model', () => 'glm-5.2'),
        },
        select: experiment.weighted({
          'claude-opus-5': 50,
          'glm-5.2': 50,
        }),
      },
    );

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
      .map((m) => ({ role: m.role, content: m.content }));

    const draft: QueryDraft = { selectedEvents: [] };

    // Memoized so it survives re-invocation: the run body re-executes after
    // every suspend (waitForEvent in validate_query, checkpoint maxRuntime),
    // and a bare Date.now() here would reset the latency clock each time.
    const startedAt = await step.run('record-start', () => Date.now());

    const result = await runAgentLoop({
      step,
      complete: (body) => chatCompletion(model, body),
      system: buildSystemPrompt({ currentQuery: clientState.currentQuery }),
      messages: [
        ...historyMessages,
        { role: 'user', content: userMessage.content },
      ],
      tools: canValidate
        ? [...insightsTools, validateQueryTool]
        : insightsTools,
      ctx: { clientState },
      draft,
      publish: (id, eventName, data) =>
        step.realtime.publish(id, ch.agent_stream, {
          event: eventName,
          data: { ...data, threadId },
          timestamp: Date.now(),
        }),
      runId,
      // Pins validate_query completions to the initiating user; empty (the
      // headless path, which never offers the tool) fails closed.
      userId: userId ?? '',
      maxIterations: 12,
    });

    const latencyMs = Date.now() - startedAt;

    await step.run('emit-scores', async () => {
      // .experiment() tags each score with its variant; runId writes them to the run span, one clean row per run.
      await inngest.score.experiment({
        name: 'query_writer_latency_ms',
        value: latencyMs,
        experiment: experimentRef,
        runId,
      });
      await inngest.score.experiment({
        name: 'query_writer_output_tokens',
        value: result.tokensOut,
        experiment: experimentRef,
        runId,
      });
      await inngest.score.experiment({
        name: 'query_writer_cost_usd',
        value: result.costUsd,
        experiment: experimentRef,
        runId,
      });
      await inngest.score({
        name: 'insights_agent_submitted',
        value: draft.sql ? 1 : 0,
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
        name: 'insights_agent_validation_attempts',
        value: result.validationAttempts,
      });
      await inngest.score({
        name: 'insights_agent_validation_failures',
        value: result.validationFailures.length,
      });
    });

    const summary =
      result.summary ||
      (draft.sql
        ? ''
        : "Sorry — I couldn't complete that request. Please try rephrasing.");

    // Fire-and-forget LLM-as-judge on every run: SQL fit when a query was
    // produced, no-query appropriateness otherwise. Attributed to the selected
    // query-writer-model variant.
    const chatContext = [
      ...(history || [])
        .map((m) => `${String(m.role ?? '')}: ${String(m.content ?? '')}`)
        .filter((line) => line.trim() !== ':'),
      `user: ${userMessage.content}`,
    ].join('\n');
    defer('judge-relevance', {
      function: insightsJudgeScorer,
      experiment: experimentRef,
      data: { sql: draft.sql ?? '', summary, chatContext },
    });

    await step.realtime.publish('publish-run-completed', ch.agent_stream, {
      event: 'run.completed',
      data: {
        threadId,
        sql: draft.sql ?? '',
        title: draft.title ?? '',
        reasoning: draft.reasoning ?? '',
        summary,
        kind: draft.sql ? 'query' : 'answer',
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
      summary,
      kind: draft.sql ? 'query' : 'answer',
      selectedEvents: draft.selectedEvents,
    };
  },
);
