'use client';

import React, {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import type { AgentStatus, ToolOutputOf } from '@inngest/use-agent';
import { useQuery } from '@tanstack/react-query';

import { useAllEventTypes } from '@/components/EventTypes/useEventTypes';
import {
  useInsightsAgent,
  type ClientState,
  type InsightsAgentConfig,
  type InsightsAgentEvent,
} from './useInsightsAgent';

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

  // Per-thread UI flags and pending SQL handoff
  getThreadFlags: (threadId: string) => ThreadFlags;
  readAndClearPendingSql: (threadId: string) => string | undefined;
  popPendingAutoRun: (threadId: string) => boolean;
  pendingSqlVersion: number; // Bumped when new SQL arrives to notify consumers

  // Client-state per thread for use in the state() function
  setThreadClientState: (threadId: string, state: ClientState) => void;

  // Event metadata for the agent
  eventTypes: string[];
  schemas: Record<string, unknown> | null;
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
  const [threadFlags, setThreadFlags] = useState<Record<string, ThreadFlags>>({});
  // Pending SQL and auto-run signals held in refs
  const pendingSqlByThreadRef = useRef<Map<string, string>>(new Map());
  const pendingAutoRunRef = useRef<Set<string>>(new Set());
  const [pendingSqlVersion, setPendingSqlVersion] = useState(0);

  // Per-thread client state map used by the state() function
  const threadClientStateRef = useRef<Map<string, ClientState>>(new Map());
  const setThreadClientState = useCallback((threadId: string, state: ClientState) => {
    threadClientStateRef.current.set(threadId, state);
  }, []);

  // Track which thread is currently sending so state() can reference the correct entry
  const activeSendThreadIdRef = useRef<string | null>(null);

  const getFlags = useCallback(
    (threadId: string): ThreadFlags => threadFlags[threadId] ?? defaultFlags,
    [threadFlags]
  );

  const readAndClearPendingSql = useCallback((threadId: string): string | undefined => {
    const sql = pendingSqlByThreadRef.current.get(threadId);
    if (sql !== undefined) {
      pendingSqlByThreadRef.current.delete(threadId);
    }
    return sql;
  }, []);

  const popPendingAutoRun = useCallback((threadId: string): boolean => {
    const has = pendingAutoRunRef.current.has(threadId);
    if (has) pendingAutoRunRef.current.delete(threadId);
    return has;
  }, []);

  const onEvent = useCallback((evt: InsightsAgentEvent) => {
    try {
      const tid = typeof evt.data.threadId === 'string' ? evt.data.threadId : undefined;
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
            if (partType === 'tool-output' && evt.data.toolName === 'generate_sql') {
              const output = evt.data.finalContent as
                | ToolOutputOf<InsightsAgentConfig, 'generate_sql'>
                | undefined;
              const sql = output?.data.sql;
              if (sql && sql.length > 0) {
                pendingSqlByThreadRef.current.set(tid, sql);
                pendingAutoRunRef.current.add(tid);
                setPendingSqlVersion((v) => v + 1);
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

  // Fetch event types and schemas once; keep it simple (no caching beyond query instance)
  // TODO: seAllEventTypes has an implicit limit of 40, need to update this to fetch more than 40 events
  const fetchAllEventTypes = useAllEventTypes();
  const { data: eventsData } = useQuery({
    queryKey: ['insights', 'all-event-types'],
    queryFn: async () => {
      const events = await fetchAllEventTypes();
      const names: string[] = events.map((e) => e.name);
      const schemaMap: Record<string, string> = {};
      for (const e of events) {
        const raw = (e.latestSchema || '').trim();
        if (!raw) continue;
        schemaMap[e.name] = raw;
      }
      return { names, schemaMap };
    },
  });

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
        schemas: eventsData?.schemaMap ?? null,
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
    [baseSendMessageToThread]
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
      readAndClearPendingSql,
      popPendingAutoRun,
      pendingSqlVersion,
      setThreadClientState,
      eventTypes: eventsData?.names ?? [],
      schemas: eventsData?.schemaMap ?? null,
    }),
    [
      messages,
      status,
      currentThreadId,
      setCurrentThreadId,
      clearThreadMessages,
      sendMessageToThread,
      getFlags,
      readAndClearPendingSql,
      popPendingAutoRun,
      pendingSqlVersion,
      setThreadClientState,
      eventsData?.names,
      eventsData?.schemaMap,
    ]
  );

  return <InsightsChatContext.Provider value={value}>{children}</InsightsChatContext.Provider>;
}

export function useInsightsChatProvider(): ContextValue {
  const ctx = useContext(InsightsChatContext);
  if (!ctx) throw new Error('useInsightsChatProvider must be used within InsightsChatProvider');
  return ctx;
}
