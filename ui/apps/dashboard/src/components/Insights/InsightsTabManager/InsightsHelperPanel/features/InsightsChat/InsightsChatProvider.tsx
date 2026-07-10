import { useQuery } from '@tanstack/react-query';
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';

import {
  useInsightsRealtime,
  sendChatMessage,
  type ClientState,
} from './useInsightsAgent';
import { useFetchInsights } from '@/components/Insights/InsightsStateMachineContext/useFetchInsights';
import { useEventTypeSchemas } from '../SchemaExplorer/SchemasContext/useEventTypeSchemas';
import type { InsightsRealtimeEvent, Message } from './types';

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

const defaultFlags: ThreadFlags = {
  networkActive: false,
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
  // Per-thread UI flags
  const [threadFlags, setThreadFlags] = useState<Record<string, ThreadFlags>>(
    {},
  );
  // Per-thread messages
  const [messagesByThread, setMessagesByThread] = useState<
    Record<string, Message[]>
  >({});
  // Current active thread
  const [currentThreadId, setCurrentThreadId] = useState<string | null>(null);
  // Latest generated SQL per thread
  const latestSqlByThreadRef = useRef<Map<string, string>>(new Map());
  const [latestSqlVersion, setLatestSqlVersion] = useState(0);

  // Per-thread client state map
  const threadClientStateRef = useRef<Map<string, ClientState>>(new Map());
  const setThreadClientState = useCallback(
    (threadId: string, state: ClientState) => {
      threadClientStateRef.current.set(threadId, state);
    },
    [],
  );

  const getFlags = useCallback(
    (threadId: string): ThreadFlags => threadFlags[threadId] ?? defaultFlags,
    [threadFlags],
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
          case 'run.started': {
            setThreadFlags((prev) => ({
              ...prev,
              [tid]: { networkActive: true },
            }));
            break;
          }

          case 'step.completed': {
            // Cache SQL when query-writer step completes
            if (evt.data.step === 'query-writer') {
              const sql =
                typeof evt.data.sql === 'string' ? evt.data.sql : undefined;
              if (sql && sql.length > 0) {
                latestSqlByThreadRef.current.set(tid, sql);
                setLatestSqlVersion((v) => v + 1);
              }
            }
            break;
          }

          case 'run.completed': {
            // Build assistant message from the completed run
            const parts: Message['parts'] = [];

            // Add SQL tool call part if present
            const sql =
              typeof evt.data.sql === 'string' ? evt.data.sql : undefined;
            if (sql) {
              parts.push({
                type: 'tool-call',
                toolName: 'generate_sql',
                data: {
                  sql,
                  title:
                    typeof evt.data.title === 'string'
                      ? evt.data.title
                      : undefined,
                  reasoning:
                    typeof evt.data.reasoning === 'string'
                      ? evt.data.reasoning
                      : undefined,
                },
              });

              // Also update the SQL cache
              latestSqlByThreadRef.current.set(tid, sql);
              setLatestSqlVersion((v) => v + 1);
            }

            // Add summary text part if present
            const summary =
              typeof evt.data.summary === 'string'
                ? evt.data.summary
                : undefined;
            if (summary) {
              parts.push({ type: 'text', content: summary });
            }

            if (parts.length > 0) {
              const assistantMsg: Message = {
                id: crypto.randomUUID(),
                role: 'assistant',
                threadId: tid,
                parts,
              };

              setMessagesByThread((prev) => ({
                ...prev,
                [tid]: [...(prev[tid] || []), assistantMsg],
              }));
            }

            setThreadFlags((prev) => ({
              ...prev,
              [tid]: { networkActive: false },
            }));
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
            // Add error as an assistant message
            const errorMessage =
              typeof evt.data.error === 'string'
                ? evt.data.error
                : 'An unknown error occurred';

            const errorMsg: Message = {
              id: crypto.randomUUID(),
              role: 'assistant',
              threadId: tid,
              parts: [{ type: 'text', content: `Error: ${errorMessage}` }],
            };

            setMessagesByThread((prev) => ({
              ...prev,
              [tid]: [...(prev[tid] || []), errorMsg],
            }));

            setThreadFlags((prev) => ({
              ...prev,
              [tid]: { networkActive: false },
            }));
            break;
          }
        }
      } catch {
        // Silently handle event processing errors
      }
    }
  }, [realtimeMessages.delta, validateSql]);

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

      setMessagesByThread((prev) => ({
        ...prev,
        [threadId]: [...(prev[threadId] || []), userMsg],
      }));

      const clientState = threadClientStateRef.current.get(threadId);

      try {
        await sendChatMessage({
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
      } catch (error) {
        // Remove the optimistic user message and show error
        setMessagesByThread((prev) => ({
          ...prev,
          [threadId]: [
            ...(prev[threadId] || []).filter((m) => m.id !== messageId),
            {
              id: crypto.randomUUID(),
              role: 'assistant' as const,
              threadId,
              parts: [
                {
                  type: 'text' as const,
                  content: `Error: ${
                    error instanceof Error
                      ? error.message
                      : 'Failed to send message'
                  }`,
                },
              ],
            },
          ],
        }));
      }
    },
    [userId, channelKey, eventsData?.names, schemas, buildHistory],
  );

  const clearThreadMessages = useCallback((threadId: string) => {
    setMessagesByThread((prev) => {
      const next = { ...prev };
      delete next[threadId];
      return next;
    });
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
