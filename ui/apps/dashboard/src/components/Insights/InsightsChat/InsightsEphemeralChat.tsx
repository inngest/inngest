'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useUser } from '@clerk/nextjs';
import {
  createInMemorySessionTransport,
  useAgents,
  type ToolCallUIPart,
} from '@inngest/use-agents';
import { v4 as uuidv4 } from 'uuid';

import type {
  GenerateSqlResult,
  SelectEventsResult,
} from '@/app/api/inngest/functions/agents/types';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';

// =============================================================================
// TOOL UI COMPONENTS
// =============================================================================

// AgentKit wraps successful tool outputs in a `data` envelope.
type ToolResultEnvelope<T> = { data: T };

function renderErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}

function GenerateSqlToolUI({
  part,
  onSqlChange,
  runQuery,
}: {
  part: ToolCallUIPart;
  onSqlChange: (sql: string) => void;
  runQuery: () => void;
}) {
  const getSuggestedSql = (toolPart: ToolCallUIPart): string | null => {
    // Guard to ensure output is accessed only when available.
    if (toolPart.state !== 'output-available') {
      return null;
    }
    const output = toolPart.output as ToolResultEnvelope<GenerateSqlResult> | undefined;
    const sql = output?.data.sql;
    if (typeof sql === 'string' && sql.trim()) {
      return sql.trim();
    }
    return null;
  };

  const suggestedSql = getSuggestedSql(part);
  const errorMessage = part.error ? renderErrorMessage(part.error) : null;

  return (
    <div className="text-xs">
      <div className="font-medium">Tool: generate_sql</div>

      {part.state === 'input-streaming' && <div className="mt-1 text-gray-600">Preparing SQL…</div>}
      {part.state === 'executing' && <div className="mt-1 text-gray-600">Generating SQL…</div>}
      {errorMessage && <div className="mt-1 text-red-600">Error: {errorMessage}</div>}

      {part.state === 'output-available' && suggestedSql && (
        <div className="mt-2 flex items-center gap-2">
          <button
            type="button"
            className="rounded border border-gray-300 bg-white px-2 py-1 text-[11px] hover:bg-gray-50"
            onClick={() => onSqlChange(suggestedSql)}
          >
            Insert SQL
          </button>
          <button
            type="button"
            className="rounded border border-gray-300 bg-white px-2 py-1 text-[11px] hover:bg-gray-50"
            onClick={() => {
              onSqlChange(suggestedSql);
              try {
                runQuery();
              } catch {}
            }}
          >
            Insert SQL & Run
          </button>
        </div>
      )}

      <div className="mt-2">
        <pre className="mt-1 overflow-auto rounded bg-white/70 p-2 text-[11px]">
          {typeof part.output === 'string' ? part.output : JSON.stringify(part.output, null, 2)}
        </pre>
      </div>
    </div>
  );
}

function SelectEventsToolUI({ part }: { part: ToolCallUIPart }) {
  const result = (part.output as ToolResultEnvelope<SelectEventsResult> | undefined)?.data;
  const selectedEvents = result?.selected;
  const errorMessage = part.error ? renderErrorMessage(part.error) : null;

  return (
    <div className="text-xs">
      <div className="font-medium">Tool: select_events</div>

      {part.state === 'input-streaming' && (
        <div className="mt-1 text-gray-600">Selecting events…</div>
      )}
      {part.state === 'executing' && <div className="mt-1 text-gray-600">Analyzing events…</div>}
      {errorMessage && <div className="mt-1 text-red-600">Error: {errorMessage}</div>}

      {part.state === 'output-available' && selectedEvents && (
        <div className="mt-2">
          <div className="font-medium text-gray-700">Selected Events:</div>
          <ul className="mt-1 list-inside list-disc rounded bg-white/70 p-2 text-[11px]">
            {selectedEvents.map((event, index) => (
              <li key={index}>
                <strong>{event.event_name}</strong>: {event.reason}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

function DefaultToolUI({ part }: { part: ToolCallUIPart }) {
  const errorMessage = part.error ? renderErrorMessage(part.error) : null;
  const outputContent =
    part.output !== undefined
      ? typeof part.output === 'string'
        ? part.output
        : JSON.stringify(part.output, null, 2)
      : null;

  return (
    <div className="text-xs">
      <div className="font-medium">Tool: {part.toolName || 'Unknown Tool'}</div>
      {part.state === 'input-streaming' && (
        <div className="mt-1 text-gray-600">Preparing input…</div>
      )}
      {part.state === 'executing' && <div className="mt-1 text-gray-600">Executing…</div>}
      {errorMessage && <div className="mt-1 text-red-600">Error: {errorMessage}</div>}
      {part.state === 'output-available' && outputContent && (
        <div className="mt-2">
          <pre className="mt-1 overflow-auto rounded bg-white/70 p-2 text-[11px]">
            {outputContent}
          </pre>
        </div>
      )}
    </div>
  );
}

// =============================================================================
// MAIN CHAT COMPONENT
// =============================================================================

type Props = {
  tabTitle: string;
  currentSql: string;
  onSqlChange: (sql: string) => void;
};

export function InsightsEphemeralChat({ tabTitle, currentSql, onSqlChange }: Props) {
  // Generate a unique thread ID for the initial chat
  const [threadId] = useState<string>(() => uuidv4());

  // State for the chat's input value
  const [inputValue, setInputValue] = useState('');

  // Get the user from the Clerk useUser hook
  const { user } = useUser();

  // Get the runQuery function from the InsightsStateMachineContext
  // Used to run the SQL query when the user clicks the "Run Query" button
  const { runQuery } = useInsightsStateMachineContext();

  // Transport: ephemeral threads, delegate network to our API
  const transport = useMemo(() => {
    // InMemorySessionTransport only delegates sendMessage and getRealtimeToken to HTTP.
    // Defaults are /api/chat and /api/realtime/token, so no config needed.
    return createInMemorySessionTransport();
  }, []);

  // Mock event catalog and schemas for the Insights network context
  const mockEventTypes = useMemo<string[]>(
    () => [
      'app/user.created',
      'app/user.updated',
      'app/user.disabled',
      'app/user.deleted',
      'session.started',
      'session.ended',
      'page.viewed',
      'purchase.completed',
      'purchase.refunded',
      'payment.failed',
      'payment.completed',
      'email.sent',
      'email.bounced',
      'feature.toggled_on',
      'feature.toggled_off',
      'error.logged',
    ],
    []
  );

  const mockSchemas = useMemo<Record<string, unknown>>(
    () => ({
      'app/user.created': {
        event_name: 'app/user.created',
        timestamp: 'DateTime',
        user_id: 'String',
        email: 'String',
        plan: 'String',
        referrer: 'String | Null',
      },
      'app/user.updated': {
        event_name: 'app/user.updated',
        timestamp: 'DateTime',
        user_id: 'String',
        changed_fields: 'Array(String)',
      },
      'app/user.disabled': {
        event_name: 'app/user.disabled',
        timestamp: 'DateTime',
        user_id: 'String',
        reason: 'String | Null',
      },
      'app/user.deleted': {
        event_name: 'app/user.deleted',
        timestamp: 'DateTime',
        user_id: 'String',
        hard_delete: 'Bool',
      },
      'session.started': {
        event_name: 'session.started',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        session_id: 'String',
        device: 'String | Null',
      },
      'session.ended': {
        event_name: 'session.ended',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        session_id: 'String',
        duration_ms: 'UInt64',
      },
      'page.viewed': {
        event_name: 'page.viewed',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        url: 'String',
        referrer: 'String | Null',
      },
      'purchase.completed': {
        event_name: 'purchase.completed',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        order_id: 'String',
        amount_cents: 'UInt64',
        currency: 'String',
      },
      'purchase.refunded': {
        event_name: 'purchase.refunded',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        order_id: 'String',
        amount_cents: 'UInt64',
        reason: 'String | Null',
      },
      'payment.failed': {
        event_name: 'payment.failed',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        attempt_id: 'String',
        code: 'String',
        message: 'String | Null',
      },
      'payment.completed': {
        event_name: 'payment.completed',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        payment_id: 'String',
        amount_cents: 'UInt64',
        method: 'String',
      },
      'email.sent': {
        event_name: 'email.sent',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        email_id: 'String',
        template: 'String | Null',
      },
      'email.bounced': {
        event_name: 'email.bounced',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        email_id: 'String',
        bounce_type: 'String',
      },
      'feature.toggled_on': {
        event_name: 'feature.toggled_on',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        feature_key: 'String',
      },
      'feature.toggled_off': {
        event_name: 'feature.toggled_off',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        feature_key: 'String',
      },
      'error.logged': {
        event_name: 'error.logged',
        timestamp: 'DateTime',
        user_id: 'String | Null',
        error_class: 'String',
        message: 'String | Null',
        stack_present: 'Bool',
      },
    }),
    []
  );

  const {
    messages,
    sendMessage,
    status,
    currentThreadId,
    setCurrentThreadId,
    clearThreadMessages,
  } = useAgents({
    enableThreadValidation: false,
    transport,
    userId: user?.id,
    channelKey: user?.id ? `insights:${user.id}` : undefined,
    state: () => ({
      sqlQuery: currentSql,
      // Insights network context for Event Matcher and Query Writer
      eventTypes: mockEventTypes,
      schemas: mockSchemas,
      currentQuery: currentSql,
      tabTitle,
      mode: 'insights_sql_playground',
      timestamp: Date.now(),
    }),
    onStateRehydrate: (messageState) => {
      if (messageState && typeof messageState === 'object') {
        const stateObj = messageState as Record<string, unknown>;
        const sql =
          typeof (stateObj as any).sql === 'string'
            ? ((stateObj as any).sql as string)
            : typeof stateObj.sqlQuery === 'string'
            ? stateObj.sqlQuery
            : typeof stateObj.currentQuery === 'string'
            ? stateObj.currentQuery
            : undefined;
        if (typeof sql === 'string' && sql !== currentSql) onSqlChange(sql);
      }
    },
  });

  // Keep per-tab thread isolated and stable
  useEffect(() => {
    if (currentThreadId !== threadId) setCurrentThreadId(threadId);
  }, [currentThreadId, setCurrentThreadId, threadId]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!inputValue.trim() || status !== 'ready') return;
      await sendMessage(inputValue);
      setInputValue('');
    },
    [inputValue, status, sendMessage]
  );

  return (
    <div className="flex h-full w-full flex-col">
      <div className="flex items-center justify-between border-b border-gray-200 px-3 py-2">
        <div className="text-sm font-medium text-gray-700">AI Assistant</div>
        <button
          className="text-xs text-gray-500 hover:text-gray-700"
          onClick={() => clearThreadMessages(threadId)}
          disabled={messages.length === 0 || status !== 'ready'}
        >
          Clear
        </button>
      </div>

      <div className="flex-1 space-y-3 overflow-auto p-3">
        {messages.map((m) => (
          <div key={m.id} className={m.role === 'user' ? 'text-right' : 'text-left'}>
            <div
              className="inline-block max-w-[340px] whitespace-pre-wrap rounded-md px-3 py-2 text-sm"
              style={{
                background: m.role === 'user' ? '#F3F4F6' : '#EEF2FF',
                color: '#111827',
              }}
            >
              {m.parts.map((p, i) => {
                if (p.type === 'text') {
                  return <div key={i}>{(p as any).content}</div>;
                }
                if (p.type === 'tool-call') {
                  const toolPart = p as ToolCallUIPart;
                  switch (toolPart.toolName) {
                    case 'generate_sql':
                      return (
                        <GenerateSqlToolUI
                          key={i}
                          part={toolPart}
                          onSqlChange={onSqlChange}
                          runQuery={runQuery}
                        />
                      );
                    case 'select_events':
                      return <SelectEventsToolUI key={i} part={toolPart} />;
                    default:
                      return <DefaultToolUI key={i} part={toolPart} />;
                  }
                }
                return null;
              })}
            </div>
          </div>
        ))}
      </div>

      <form onSubmit={handleSubmit} className="border-t border-gray-200 p-2">
        <input
          className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm outline-none focus:border-gray-400"
          placeholder="Ask anything…"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          disabled={status !== 'ready'}
        />
      </form>
    </div>
  );
}
