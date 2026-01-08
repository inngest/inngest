import type { AgentStatus, ToolOutputOf } from '@inngest/use-agent';
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
  useInsightsAgent,
  type ClientState,
  type InsightsAgentConfig,
  type InsightsAgentEvent,
} from './useInsightsAgent';
import { useEventTypeSchemas } from '../SchemaExplorer/SchemasContext/useEventTypeSchemas';

type ThreadFlags = {
  networkActive: boolean;
  textStreaming: boolean;
  textCompleted: boolean;
  currentToolName: string | null;
};

type ContextValue = {
  // Core from useAgents
  messages: ReturnType<typeof useInsightsAgent>['messages'];
  status: AgentStatus;
  currentThreadId: string | null;
  setCurrentThreadId: (id: string) => void;
  clearThreadMessages: (threadId: string) => void;
  // Wrapped send to associate per-thread client state
  sendMessageToThread: (threadId: string, content: string) => Promise<void>;

  // Per-thread UI flags and derived SQL
  getThreadFlags: (threadId: string) => ThreadFlags;
  getLatestGeneratedSql: (threadId: string) => string | undefined;
  latestSqlVersion: number; // Bumped when new SQL arrives to notify consumers

  // Client-state per thread for use in the state() function
  setThreadClientState: (threadId: string, state: ClientState) => void;

  // Event metadata for the agent
  eventTypes: string[];
  schemas: { name: string; schema: string }[];
};

const defaultFlags: ThreadFlags = {
  networkActive: false,
  textStreaming: false,
  textCompleted: false,
  currentToolName: null,
};

const InsightsChatContext = createContext<ContextValue | undefined>(undefined);

export function InsightsChatProvider({ children }: { children: ReactNode }) {
  // Per-thread UI flags in React state for rerenders
  const [threadFlags, setThreadFlags] = useState<Record<string, ThreadFlags>>(
    {},
  );
  // Latest generated SQL per thread
  const latestSqlByThreadRef = useRef<Map<string, string>>(new Map());
  const [latestSqlVersion, setLatestSqlVersion] = useState(0);

  // Per-thread client state map used by the state() function
  const threadClientStateRef = useRef<Map<string, ClientState>>(new Map());
  const setThreadClientState = useCallback(
    (threadId: string, state: ClientState) => {
      threadClientStateRef.current.set(threadId, state);
    },
    [],
  );

  // Track which thread is currently sending so state() can reference the correct entry
  const activeSendThreadIdRef = useRef<string | null>(null);

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

  const onEvent = useCallback((evt: InsightsAgentEvent) => {
    try {
      const tid =
        typeof evt.data.threadId === 'string' ? evt.data.threadId : undefined;
      if (!tid) return;

      setThreadFlags((prev) => {
        const prevFlags = prev[tid] ?? defaultFlags;
        switch (evt.event) {
          case 'run.started': {
            return {
              ...prev,
              [tid]: {
                networkActive: true,
                textStreaming: false,
                textCompleted: false,
                currentToolName: null,
              },
            };
          }
          case 'text.delta': {
            return {
              ...prev,
              [tid]: {
                ...prevFlags,
                textStreaming: true,
                textCompleted: false,
              },
            };
          }
          case 'tool_call.arguments.delta': {
            // evt is narrowed by the discriminant here
            const toolName = evt.data.toolName;
            return {
              ...prev,
              [tid]: {
                ...prevFlags,
                currentToolName:
                  typeof toolName === 'string' && toolName.length > 0
                    ? toolName
                    : prevFlags.currentToolName,
              },
            };
          }
          case 'part.completed': {
            // evt is narrowed by the discriminant here
            const partType = evt.data.type;
            // Clear tool name once tool step completes
            const nextFlags: ThreadFlags = {
              ...prevFlags,
              currentToolName:
                partType === 'tool-output' || partType === 'tool-call'
                  ? null
                  : prevFlags.currentToolName,
            };

            // If text part completes, mark completion
            if (partType === 'text') {
              nextFlags.textStreaming = false;
              nextFlags.textCompleted = true;
            }

            // Capture generated SQL from tool-output (typed via manifest)
            if (
              partType === 'tool-output' &&
              evt.data.toolName === 'generate_sql'
            ) {
              const output = evt.data.finalContent as
                | ToolOutputOf<InsightsAgentConfig, 'generate_sql'>
                | undefined;
              const sql = output?.data.sql;
              if (sql && sql.length > 0) {
                latestSqlByThreadRef.current.set(tid, sql);
                setLatestSqlVersion((v) => v + 1);
              }
            }

            return {
              ...prev,
              [tid]: nextFlags,
            };
          }
          case 'stream.ended': {
            return {
              ...prev,
              [tid]: {
                networkActive: false,
                textStreaming: false,
                textCompleted: true,
                currentToolName: null,
              },
            };
          }
          default:
            return prev;
        }
      });
    } catch {}
  }, []);

  // Fetch event types and schemas using the same hook as SchemaExplorer
  const getEventTypeSchemas = useEventTypeSchemas();
  const { data: eventsData } = useQuery({
    queryKey: ['insights', 'all-event-types'],
    queryFn: async () => {
      // Fetch up to 5 pages (200 events max)
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

          // Check if there are more pages
          if (result.pageInfo.hasNextPage && result.pageInfo.endCursor) {
            cursor = result.pageInfo.endCursor;
          } else {
            break;
          }
        }
      } catch (error) {
        console.error('Failed to fetch event type schemas:', error);
        // Return partial data if some pages were fetched successfully
        // This ensures the UI remains functional even if pagination fails
      }

      return { names, schemaMap };
    },
  });

  // Convert schemaMap to schemas array (memoized to avoid recomputation)
  const schemas = useMemo(() => {
    const schemaMap = eventsData?.schemaMap ?? {};
    return Object.entries(schemaMap).map(([name, schema]) => ({
      name,
      schema,
    }));
  }, [eventsData?.schemaMap]);

  const {
    messages,
    status,
    currentThreadId,
    setCurrentThreadId,
    clearThreadMessages,
    sendMessageToThread: baseSendMessageToThread,
  } = useInsightsAgent({
    enableThreadValidation: false,
    state: () => {
      const tid = activeSendThreadIdRef.current;
      if (tid) {
        const s = threadClientStateRef.current.get(tid);
        if (s) return s;
      }
      // Fallback minimal state
      return {
        sqlQuery: '',
        eventTypes: eventsData?.names ?? [],
        schemas,
        currentQuery: '',
        tabTitle: '',
        mode: 'insights_sql_playground',
        timestamp: Date.now(),
      } as ClientState;
    },
    onEvent,
  });

  const sendMessageToThread = useCallback(
    async (threadId: string, content: string) => {
      try {
        activeSendThreadIdRef.current = threadId;
        await baseSendMessageToThread(threadId, content);
      } finally {
        activeSendThreadIdRef.current = null;
      }
    },
    [baseSendMessageToThread],
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
