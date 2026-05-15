import { anthropic } from 'inngest';
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
  channelKey?: string;
  history?: Array<Record<string, unknown>>;
};

export const runInsightsAgent = inngest.createFunction(
  {
    id: 'run-insights-agent',
    name: 'Insights SQL Agent',
    triggers: [{ event: 'insights-agent/chat.requested' }],
  },
  async ({ event, step }) => {
    const {
      threadId: providedThreadId,
      userMessage,
      userId,
      channelKey,
      history,
    } = event.data as ChatEventData;

    if (!userId) {
      throw new Error('userId is required for agent chat execution');
    }

    const threadId = await step.run('generate-thread-id', () => {
      return providedThreadId || uuidv4();
    });

    const targetChannel = await step.run('generate-target-channel', () => {
      return channelKey || userId;
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

    const queryWriterResult = await step.ai.infer('query-writer', {
      model: anthropic({
        model: 'claude-sonnet-4-5',
        defaultParameters: { max_tokens: 4096 },
      }),
      body: {
        system: queryWriterPrompt.system,
        messages: queryWriterPrompt.messages,
        tools: [generateSqlTool],
        tool_choice: { type: 'tool' as const, name: 'generate_sql' },
      },
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
