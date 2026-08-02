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
  fetchRunResult,
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

// Realtime has no replay, so a run that finishes while the browser is
// disconnected leaves the thread spinning on a run.completed that will never
// arrive. Each in-flight run gets a watchdog: after this long with no traffic
// for the thread, read the result back from the run itself.
const IDLE_BEFORE_RECONCILE_MS = 45_000;
// The runs endpoint caches for 15s, so polling faster only re-reads the cache.
const RECONCILE_INTERVAL_MS = 20_000;
const MAX_RECONCILE_ATTEMPTS = 8;

type PendingRun = { eventId: string; attempts: number; timer: number };

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

  // Runs whose result we haven't seen yet, keyed by thread.
  const pendingRunsRef = useRef<Map<string, PendingRun>>(new Map());

  // Threads awaiting a result, from send until the first terminal outcome
  // lands. Realtime and the run-output fallback can both deliver the same
  // result, so whichever arrives first clears this and the other one no-ops.
  // Kept separate from pendingRunsRef, which only exists when /api/chat
  // returned an event id: gating on that would silently drop results.
  const awaitingResultRef = useRef<Set<string>>(new Set());

  const clearPending = useCallback((threadId: string) => {
    const pending = pendingRunsRef.current.get(threadId);
    if (pending) window.clearTimeout(pending.timer);
    pendingRunsRef.current.delete(threadId);
  }, []);

  // Terminal state for a thread, whichever channel delivered it.
  const finishThread = useCallback(
    (threadId: string, parts: Message['parts']) => {
      awaitingResultRef.current.delete(threadId);

      if (parts.length > 0) {
        const assistantMsg: Message = {
          id: crypto.randomUUID(),
          role: 'assistant',
          threadId,
          parts,
        };

        setMessagesByThread((prev) => ({
          ...prev,
          [threadId]: [...(prev[threadId] || []), assistantMsg],
        }));
      }

      setThreadFlags((prev) => ({
        ...prev,
        [threadId]: { networkActive: false },
      }));
      clearPending(threadId);
    },
    [clearPending],
  );

  // Shared by the realtime run.completed event and the run-output fallback,
  // which carry the same payload.
  const applyRunResult = useCallback(
    (threadId: string, data: Record<string, unknown>) => {
      if (!awaitingResultRef.current.has(threadId)) return;

      const parts: Message['parts'] = [];

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

        latestSqlByThreadRef.current.set(threadId, sql);
        setLatestSqlVersion((v) => v + 1);
      }

      const summary =
        typeof data.summary === 'string' ? data.summary : undefined;
      if (summary) {
        parts.push({ type: 'text', content: summary });
      }

      finishThread(threadId, parts);
    },
    [finishThread],
  );

  const failThread = useCallback(
    (threadId: string, message: string) => {
      if (!awaitingResultRef.current.has(threadId)) return;
      finishThread(threadId, [{ type: 'text', content: `Error: ${message}` }]);
    },
    [finishThread],
  );

  // Indirection so the watchdog can reschedule itself without the callbacks
  // forming a cycle.
  const reconcileRef = useRef<(threadId: string) => void>(() => {});

  const scheduleReconcile = useCallback((threadId: string, delayMs: number) => {
    const pending = pendingRunsRef.current.get(threadId);
    if (!pending) return;
    window.clearTimeout(pending.timer);
    pending.timer = window.setTimeout(
      () => reconcileRef.current(threadId),
      delayMs,
    );
  }, []);

  const reconcile = useCallback(
    async (threadId: string) => {
      const pending = pendingRunsRef.current.get(threadId);
      if (!pending) return;
      pending.attempts += 1;

      const result = await fetchRunResult(pending.eventId, threadId).catch(
        () => null,
      );
      // Identity, not presence: run.completed may have landed while the request
      // was in flight, and the thread may already be waiting on a newer run.
      if (pendingRunsRef.current.get(threadId) !== pending) return;

      if (result?.status === 'Completed' && result.output) {
        applyRunResult(threadId, result.output);
        return;
      }

      if (result?.status === 'Failed' || result?.status === 'Cancelled') {
        failThread(
          threadId,
          'The Insights agent could not complete this request. Please try again.',
        );
        return;
      }

      if (pending.attempts >= MAX_RECONCILE_ATTEMPTS) {
        failThread(
          threadId,
          "This request didn't come back. Please try sending it again.",
        );
        return;
      }

      scheduleReconcile(threadId, RECONCILE_INTERVAL_MS);
    },
    [applyRunResult, failThread, scheduleReconcile],
  );

  useEffect(() => {
    reconcileRef.current = (threadId: string) => void reconcile(threadId);
  }, [reconcile]);

  // Reconnecting means we were away while the run may have finished; don't
  // wait out the watchdog to find out.
  const prevConnectionStatusRef = useRef(connectionStatus);
  useEffect(() => {
    const prev = prevConnectionStatusRef.current;
    prevConnectionStatusRef.current = connectionStatus;

    if (
      connectionStatus === 'open' &&
      (prev === 'closed' || prev === 'error' || prev === 'paused')
    ) {
      for (const threadId of pendingRunsRef.current.keys()) {
        reconcileRef.current(threadId);
      }
    }
  }, [connectionStatus]);

  useEffect(() => {
    const pendingRuns = pendingRunsRef.current;
    return () => {
      for (const pending of pendingRuns.values()) {
        window.clearTimeout(pending.timer);
      }
      pendingRuns.clear();
    };
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

      // Any traffic for this thread means the run is alive; push the watchdog out.
      if (pendingRunsRef.current.has(tid)) {
        scheduleReconcile(tid, IDLE_BEFORE_RECONCILE_MS);
      }

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
            // Add error as an assistant message
            const errorMessage =
              typeof evt.data.error === 'string'
                ? evt.data.error
                : 'An unknown error occurred';

            failThread(tid, errorMessage);
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
    scheduleReconcile,
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

      setMessagesByThread((prev) => ({
        ...prev,
        [threadId]: [...(prev[threadId] || []), userMsg],
      }));

      const clientState = threadClientStateRef.current.get(threadId);

      // Marked before the await: a fast run could publish run.completed before
      // this request even returns.
      awaitingResultRef.current.add(threadId);

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

        if (eventId) {
          pendingRunsRef.current.set(threadId, {
            eventId,
            attempts: 0,
            timer: 0,
          });
          scheduleReconcile(threadId, IDLE_BEFORE_RECONCILE_MS);
        }
      } catch (error) {
        awaitingResultRef.current.delete(threadId);

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
    [
      userId,
      channelKey,
      eventsData?.names,
      schemas,
      buildHistory,
      scheduleReconcile,
    ],
  );

  const clearThreadMessages = useCallback(
    (threadId: string) => {
      setMessagesByThread((prev) => {
        const next = { ...prev };
        delete next[threadId];
        return next;
      });
      latestSqlByThreadRef.current.delete(threadId);
      // Don't let a discarded thread's run report back into the cleared view,
      // on either channel.
      awaitingResultRef.current.delete(threadId);
      clearPending(threadId);
    },
    [clearPending],
  );

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
