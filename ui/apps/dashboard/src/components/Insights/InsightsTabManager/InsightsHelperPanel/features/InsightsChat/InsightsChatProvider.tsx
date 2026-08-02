import { useQuery } from '@tanstack/react-query';
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  useState,
  type ReactNode,
} from 'react';

import { assistantMessage, chatReducer, initialChatState } from './chatReducer';
import { RunResultWatcher } from './RunResultWatcher';
import {
  useInsightsRealtime,
  sendChatMessage,
  type ClientState,
} from './useInsightsAgent';
import { useFetchInsights } from '@/components/Insights/InsightsStateMachineContext/useFetchInsights';
import { useEventTypeSchemas } from '../SchemaExplorer/SchemasContext/useEventTypeSchemas';
import type { InsightsRealtimeEvent, Message, MessagePart } from './types';

type ThreadFlags = {
  networkActive: boolean;
};

type ContextValue = {
  // Messages for the current thread
  messages: Message[];
  status: 'ready' | 'loading';
  currentThreadId: string | null;
  setCurrentThreadId: (id: string) => void;
  clearThreadMessages: (threadId: string) => void;
  // Wrapped send to associate per-thread client state
  sendMessageToThread: (threadId: string, content: string) => Promise<void>;

  // Per-thread UI flags and derived SQL
  getThreadFlags: (threadId: string) => ThreadFlags;
  getLatestGeneratedSql: (threadId: string) => string | undefined;
  latestSqlVersion: number;

  // Client-state per thread
  setThreadClientState: (threadId: string, state: ClientState) => void;

  // Event metadata for the agent
  eventTypes: string[];
  schemas: { name: string; schema: string }[];
};

const InsightsChatContext = createContext<ContextValue | undefined>(undefined);

export function InsightsChatProvider({
  userId,
  channelKey,
  children,
}: {
  userId?: string;
  channelKey?: string;
  children: ReactNode;
}) {
  const [{ messagesByThread, pendingByThread }, dispatch] = useReducer(
    chatReducer,
    initialChatState,
  );
  // Current active thread
  const [currentThreadId, setCurrentThreadId] = useState<string | null>(null);
  // Latest generated SQL per thread
  const latestSqlByThreadRef = useRef<Map<string, string>>(new Map());
  const [latestSqlVersion, setLatestSqlVersion] = useState(0);

  // Bumping the version re-runs the editor's auto-apply, so only publish SQL
  // we haven't already seen for this thread. The query-writer step and the
  // completed run report the same SQL.
  const cacheSql = useCallback((threadId: string, sql: string) => {
    if (latestSqlByThreadRef.current.get(threadId) === sql) return;
    latestSqlByThreadRef.current.set(threadId, sql);
    setLatestSqlVersion((v) => v + 1);
  }, []);

  // Per-thread client state map
  const threadClientStateRef = useRef<Map<string, ClientState>>(new Map());
  const setThreadClientState = useCallback(
    (threadId: string, state: ClientState) => {
      threadClientStateRef.current.set(threadId, state);
    },
    [],
  );

  const getFlags = useCallback(
    (threadId: string): ThreadFlags => ({
      networkActive: threadId in pendingByThread,
    }),
    [pendingByThread],
  );

  const getLatestGeneratedSql = useCallback(
    (threadId: string): string | undefined => {
      return latestSqlByThreadRef.current.get(threadId);
    },
    [],
  );

  // Realtime subscription
  const { messages: realtimeMessages, connectionStatus } = useInsightsRealtime({
    channelKey,
    enabled: !!channelKey,
  });

  // The browser half of the agent's validate_query round trip: run the
  // candidate SQL with this user's credentials, then report the outcome so
  // the waiting agent run can self-correct.
  const { fetchInsights } = useFetchInsights();
  const validateSql = useCallback(
    async (validationId: string, sql: string) => {
      let result: {
        ok: boolean;
        columns?: string[];
        rowCount?: number;
        diagnostics?: { code?: string; message: string }[];
      };
      try {
        const res = await fetchInsights(
          { query: sql, queryName: 'agent-validation' },
          () => {},
        );
        const errors = res.diagnostics.filter((d) => d.severity === 'error');
        result =
          errors.length > 0
            ? {
                ok: false,
                diagnostics: errors.map((d) => ({
                  code: d.code,
                  message: d.message,
                })),
              }
            : {
                ok: true,
                columns: res.columns.map((c) => c.name),
                rowCount: res.rows.length,
              };
      } catch (error) {
        result = {
          ok: false,
          diagnostics: [
            {
              message: error instanceof Error ? error.message : 'Query failed',
            },
          ],
        };
      }

      try {
        await fetch('/api/chat-validate', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ validationId, ...result }),
        });
      } catch {
        // Nothing to do — the agent times out and proceeds unvalidated.
      }
    },
    [fetchInsights],
  );

  // Derive loading status from connection state
  const status: 'ready' | 'loading' = useMemo(() => {
    if (connectionStatus === 'connecting') return 'loading';
    return 'ready';
  }, [connectionStatus]);

  // Shared by the realtime run.completed event and the recovery poll, which
  // carry the same payload.
  const applyRunResult = useCallback(
    (threadId: string, data: Record<string, unknown>) => {
      const parts: MessagePart[] = [];

      const sql = typeof data.sql === 'string' ? data.sql : undefined;
      if (sql) {
        parts.push({
          type: 'tool-call',
          toolName: 'generate_sql',
          data: {
            sql,
            title: typeof data.title === 'string' ? data.title : undefined,
            reasoning:
              typeof data.reasoning === 'string' ? data.reasoning : undefined,
          },
        });

        cacheSql(threadId, sql);
      }

      const summary =
        typeof data.summary === 'string' ? data.summary : undefined;
      if (summary) parts.push({ type: 'text', content: summary });

      dispatch({
        type: 'result',
        threadId,
        message: parts.length > 0 ? assistantMessage(threadId, parts) : null,
      });
    },
    [cacheSql],
  );

  const failThread = useCallback((threadId: string, message: string) => {
    dispatch({
      type: 'result',
      threadId,
      message: assistantMessage(threadId, [
        { type: 'text', content: `Error: ${message}` },
      ]),
    });
  }, []);

  // Process new realtime events
  useEffect(() => {
    for (const msg of realtimeMessages.delta) {
      if (msg.kind !== 'data' || msg.topic !== 'agent_stream') continue;

      const evt = msg.data as InsightsRealtimeEvent | undefined;
      if (!evt || typeof evt.event !== 'string') continue;
      const tid =
        typeof evt.data.threadId === 'string' ? evt.data.threadId : undefined;
      if (!tid) continue;

      try {
        switch (evt.event) {
          case 'step.completed': {
            // Cache SQL when query-writer step completes
            if (evt.data.step === 'query-writer') {
              const sql =
                typeof evt.data.sql === 'string' ? evt.data.sql : undefined;
              if (sql && sql.length > 0) cacheSql(tid, sql);
            }
            break;
          }

          case 'run.completed': {
            applyRunResult(tid, evt.data);
            break;
          }

          case 'validation.requested': {
            const validationId =
              typeof evt.data.validationId === 'string'
                ? evt.data.validationId
                : undefined;
            const sql =
              typeof evt.data.sql === 'string' ? evt.data.sql : undefined;
            if (validationId && sql) void validateSql(validationId, sql);
            break;
          }

          case 'error': {
            failThread(
              tid,
              typeof evt.data.error === 'string'
                ? evt.data.error
                : 'An unknown error occurred',
            );
            break;
          }
        }
      } catch {
        // Silently handle event processing errors
      }
    }
  }, [
    realtimeMessages.delta,
    validateSql,
    applyRunResult,
    failThread,
    cacheSql,
  ]);

  // Fetch event types and schemas using the same hook as SchemaExplorer
  const getEventTypeSchemas = useEventTypeSchemas();
  const { data: eventsData } = useQuery({
    queryKey: ['insights', 'all-event-types'],
    queryFn: async () => {
      const MAX_PAGES = 5;
      let cursor: string | null = null;
      const names: string[] = [];
      const schemaMap: Record<string, string> = {};

      try {
        for (let i = 0; i < MAX_PAGES; i++) {
          const result = await getEventTypeSchemas({
            cursor,
            nameSearch: null,
          });

          for (const event of result.events) {
            names.push(event.name);
            const raw = (event.schema || '').trim();
            if (raw) {
              schemaMap[event.name] = raw;
            }
          }

          if (result.pageInfo.hasNextPage && result.pageInfo.endCursor) {
            cursor = result.pageInfo.endCursor;
          } else {
            break;
          }
        }
      } catch (error) {
        console.error('Failed to fetch event type schemas:', error);
      }

      return { names, schemaMap };
    },
  });

  const schemas = useMemo(() => {
    const schemaMap = eventsData?.schemaMap ?? {};
    return Object.entries(schemaMap).map(([name, schema]) => ({
      name,
      schema,
    }));
  }, [eventsData?.schemaMap]);

  // Build conversation history for the backend
  const buildHistory = useCallback(
    (threadId: string): Array<Record<string, unknown>> => {
      const msgs = messagesByThread[threadId] || [];
      return msgs.flatMap((msg) =>
        msg.parts
          .filter((part) => part.type === 'text')
          .map((part) => ({
            role: msg.role,
            type: 'text',
            content: (part as { content: string }).content,
          })),
      );
    },
    [messagesByThread],
  );

  const sendMessageToThread = useCallback(
    async (threadId: string, content: string) => {
      if (!userId) return;

      const messageId = crypto.randomUUID();
      const userMsg: Message = {
        id: messageId,
        role: 'user',
        threadId,
        parts: [{ type: 'text', content }],
      };

      // Marks the thread pending, which is what turns the spinner on. Done
      // before the await so a run that answers immediately still has a thread
      // to answer into.
      dispatch({ type: 'send', threadId, message: userMsg });

      const clientState = threadClientStateRef.current.get(threadId);

      try {
        const { eventId } = await sendChatMessage({
          content,
          messageId,
          threadId,
          userId,
          channelKey,
          state: clientState
            ? {
                eventTypes: clientState.eventTypes,
                schemas: clientState.schemas,
                currentQuery: clientState.currentQuery,
              }
            : {
                eventTypes: eventsData?.names ?? [],
                schemas,
              },
          history: buildHistory(threadId),
        });

        // Lets RunResultWatcher recover the result if the realtime
        // run.completed never reaches the browser.
        if (eventId) dispatch({ type: 'sent', threadId, eventId });
      } catch (error) {
        // Remove the optimistic user message and show error
        dispatch({
          type: 'sendFailed',
          threadId,
          messageId,
          message: assistantMessage(threadId, [
            {
              type: 'text',
              content: `Error: ${
                error instanceof Error
                  ? error.message
                  : 'Failed to send message'
              }`,
            },
          ]),
        });
      }
    },
    [userId, channelKey, eventsData?.names, schemas, buildHistory],
  );

  const clearThreadMessages = useCallback((threadId: string) => {
    dispatch({ type: 'clearThread', threadId });
    latestSqlByThreadRef.current.delete(threadId);
  }, []);

  const messages = useMemo(
    () => messagesByThread[currentThreadId || ''] || [],
    [messagesByThread, currentThreadId],
  );

  const value: ContextValue = useMemo(
    () => ({
      messages,
      status,
      currentThreadId,
      setCurrentThreadId,
      clearThreadMessages,
      sendMessageToThread,
      getThreadFlags: getFlags,
      getLatestGeneratedSql,
      latestSqlVersion,
      setThreadClientState,
      eventTypes: eventsData?.names ?? [],
      schemas,
    }),
    [
      messages,
      status,
      currentThreadId,
      setCurrentThreadId,
      clearThreadMessages,
      sendMessageToThread,
      getFlags,
      getLatestGeneratedSql,
      latestSqlVersion,
      setThreadClientState,
      eventsData?.names,
      schemas,
    ],
  );

  return (
    <InsightsChatContext.Provider value={value}>
      {Object.entries(pendingByThread).map(([threadId, eventId]) =>
        eventId ? (
          <RunResultWatcher
            key={eventId}
            eventId={eventId}
            threadId={threadId}
            onResult={applyRunResult}
            onFail={failThread}
          />
        ) : null,
      )}
      {children}
    </InsightsChatContext.Provider>
  );
}

export function useInsightsChatProvider(): ContextValue {
  const ctx = useContext(InsightsChatContext);
  if (!ctx)
    throw new Error(
      'useInsightsChatProvider must be used within InsightsChatProvider',
    );
  return ctx;
}
