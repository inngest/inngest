import { useQuery } from '@tanstack/react-query';
import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';

import {
  attachRun,
  clearThread,
  isThreadWaiting,
  pendingRuns,
  settleTurn,
  startTurn,
  type Threads,
} from './chatMessages';
import { RunWatcher } from './RunWatcher';
import { sendChatMessage, type ClientState } from './useInsightsAgent';
import { useEventTypeSchemas } from '../SchemaExplorer/SchemasContext/useEventTypeSchemas';
import type { Message, MessagePart } from './types';

type ThreadFlags = {
  networkActive: boolean;
};

type ContextValue = {
  // Messages for the current thread
  messages: Message[];
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

function errorParts(message: string): MessagePart[] {
  return [{ type: 'text', content: `Error: ${message}` }];
}

export function InsightsChatProvider({
  userId,
  channelKey,
  children,
}: {
  userId?: string;
  channelKey?: string;
  children: ReactNode;
}) {
  // Per-thread messages, and the only chat state: a turn in flight is a
  // message, so nothing else has to track what's outstanding.
  const [threads, setThreads] = useState<Threads>({});
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
    (threadId: string): ThreadFlags => ({
      networkActive: isThreadWaiting(threads, threadId),
    }),
    [threads],
  );

  const getLatestGeneratedSql = useCallback(
    (threadId: string): string | undefined => {
      return latestSqlByThreadRef.current.get(threadId);
    },
    [],
  );

  // Bumping the version is what re-runs the editor's auto-apply.
  const applyRunResult = useCallback(
    (threadId: string, turnId: string, output: Record<string, unknown>) => {
      const parts: MessagePart[] = [];

      const sql = typeof output.sql === 'string' ? output.sql : undefined;
      if (sql) {
        parts.push({
          type: 'tool-call',
          toolName: 'generate_sql',
          data: {
            sql,
            title: typeof output.title === 'string' ? output.title : undefined,
            reasoning:
              typeof output.reasoning === 'string'
                ? output.reasoning
                : undefined,
          },
        });

        latestSqlByThreadRef.current.set(threadId, sql);
        setLatestSqlVersion((v) => v + 1);
      }

      const summary =
        typeof output.summary === 'string' ? output.summary : undefined;
      if (summary) parts.push({ type: 'text', content: summary });

      setThreads((prev) => settleTurn(prev, threadId, turnId, parts));
    },
    [],
  );

  const failTurn = useCallback(
    (threadId: string, turnId: string, message: string) => {
      setThreads((prev) =>
        settleTurn(prev, threadId, turnId, errorParts(message)),
      );
    },
    [],
  );

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
      const msgs = threads[threadId] || [];
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
    [threads],
  );

  const sendMessageToThread = useCallback(
    async (threadId: string, content: string) => {
      if (!userId) return;

      const messageId = crypto.randomUUID();
      const turnId = crypto.randomUUID();
      const userMsg: Message = {
        id: messageId,
        role: 'user',
        threadId,
        parts: [{ type: 'text', content }],
      };

      // The waiting turn goes in before the await, so the spinner is on from
      // the moment the user sends rather than from whenever the run reports in.
      setThreads((prev) => startTurn(prev, threadId, userMsg, turnId));

      const clientState = threadClientStateRef.current.get(threadId);

      try {
        const { eventId, receipt } = await sendChatMessage({
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

        if (!eventId || !receipt) {
          // Without a run to read there's nothing to wait for.
          failTurn(threadId, turnId, 'The request could not be tracked.');
          return;
        }

        setThreads((prev) =>
          attachRun(prev, threadId, turnId, { eventId, receipt }),
        );
      } catch (error) {
        failTurn(
          threadId,
          turnId,
          error instanceof Error ? error.message : 'Failed to send message',
        );
      }
    },
    [userId, channelKey, eventsData?.names, schemas, buildHistory, failTurn],
  );

  const clearThreadMessages = useCallback((threadId: string) => {
    setThreads((prev) => clearThread(prev, threadId));
    latestSqlByThreadRef.current.delete(threadId);
  }, []);

  const messages = useMemo(
    () => threads[currentThreadId || ''] || [],
    [threads, currentThreadId],
  );

  const watched = useMemo(() => pendingRuns(threads), [threads]);

  const value: ContextValue = useMemo(
    () => ({
      messages,
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
      {watched.map(({ threadId, turnId, run }) => (
        <RunWatcher
          key={turnId}
          run={run}
          threadId={threadId}
          turnId={turnId}
          onResult={applyRunResult}
          onFail={failTurn}
        />
      ))}
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
