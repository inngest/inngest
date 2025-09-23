'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { type AgentStatus } from '@inngest/use-agents';

import { useAllEventTypes } from '@/components/EventTypes/useEventTypes';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Conversation, ConversationContent } from './Conversation';
import { useInsightsChatProvider } from './InsightsChatProvider';
import { LoadingIndicator } from './LoadingIndicator';
import { ChatHeader } from './header/ChatHeader';
import { ResponsivePromptInput } from './input/InputField';
import { AssistantMessage } from './messages/AssistantMessage';
import { ToolMessage } from './messages/ToolMessage';
import { UserMessage } from './messages/UserMessage';

// Types for derived event data
type Schemas = Record<string, unknown>;
type EventTypes = string[];
type AllEventType = { id: string; name: string; latestSchema: string };

// Helper: derive dynamic loading text from event-driven flags
function getLoadingMessage(flags: {
  networkActive: boolean;
  textStreaming: boolean;
  textCompleted: boolean;
  toolName?: string | null;
  status: AgentStatus;
}): string | null {
  const { networkActive, textStreaming, textCompleted, toolName } = flags;
  if (!networkActive) return null;
  if (textStreaming) return null;
  if (textCompleted) return null;
  if (toolName) {
    switch (toolName) {
      case 'select_events':
        return 'Analyzing events…';
      case 'generate_sql':
        return 'Generating query...';
      default:
        return 'Thinking...';
    }
  }
  return 'Thinking…';
}

export function InsightsChat({
  threadId,
  onToggleChat,
  className,
}: {
  threadId: string;
  onToggleChat: () => void;
  className?: string;
}) {
  // Read required data from the Insights state context
  const {
    query: currentSql,
    queryName: tabTitle,
    onChange: onSqlChange,
    runQuery,
  } = useInsightsStateMachineContext();

  // State for the chat's input value
  const [inputValue, setInputValue] = useState('');

  // Load event types and schemas via GraphQL-backed hook
  const fetchAllEventTypes = useAllEventTypes();
  const [schemas, setSchemas] = useState<Schemas | null>(null);
  const [eventTypes, setEventTypes] = useState<EventTypes>([]);

  useEffect(() => {
    (async () => {
      try {
        const events: AllEventType[] = await fetchAllEventTypes();
        const names: EventTypes = events.map((e) => e.name);
        const schemaMap: Schemas = {};
        for (const e of events) {
          const raw = e.latestSchema.trim();
          if (!raw) continue;
          try {
            schemaMap[e.name] = JSON.parse(raw);
          } catch {
            schemaMap[e.name] = raw;
          }
        }
        setEventTypes(names);
        setSchemas(schemaMap);
      } catch {
        setEventTypes([]);
        setSchemas(null);
      }
    })();
  }, [fetchAllEventTypes]);

  // Provider-backed agent state and actions
  const {
    messages,
    status,
    currentThreadId,
    setCurrentThreadId,
    clearThreadMessages,
    sendMessageToThread,
    getThreadFlags,
    readAndClearPendingSql,
    popPendingAutoRun,
    pendingSqlVersion,
    setThreadClientState,
  } = useInsightsChatProvider();

  // Derive loading flags for this thread from provider
  const { networkActive, textStreaming, textCompleted, currentToolName } = useMemo(
    () => getThreadFlags(threadId),
    [getThreadFlags, threadId]
  );

  // Keep per-tab thread isolated and stable
  useEffect(() => {
    if (currentThreadId !== threadId) setCurrentThreadId(threadId);
  }, [currentThreadId, setCurrentThreadId, threadId]);

  // Keep provider's per-thread client state up to date
  useEffect(() => {
    setThreadClientState(threadId, {
      sqlQuery: currentSql,
      eventTypes,
      schemas,
      currentQuery: currentSql,
      tabTitle,
      mode: 'insights_sql_playground',
      timestamp: Date.now(),
    });
  }, [setThreadClientState, threadId, currentSql, eventTypes, schemas, tabTitle]);

  // Apply pending SQL from background tool-output when becoming active, then optionally auto-run
  const lastAppliedSqlRef = useRef<string | null>(null);
  useEffect(() => {
    // Only act for this thread when it's the active one
    if (currentThreadId !== threadId) return;
    const sql = readAndClearPendingSql(threadId);
    if (typeof sql === 'string' && sql.length > 0) {
      lastAppliedSqlRef.current = sql;
      onSqlChange(sql.trim());
    }
    if (popPendingAutoRun(threadId)) {
      // Defer run slightly to allow onSqlChange to commit
      setTimeout(() => {
        runQuery();
      }, 0);
    }
    // Re-run when provider reports new pending SQL ingress
  }, [
    currentThreadId,
    threadId,
    readAndClearPendingSql,
    popPendingAutoRun,
    onSqlChange,
    runQuery,
    pendingSqlVersion,
  ]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const message = inputValue.trim();
      if (!message || status !== 'ready') return;
      // Clear input immediately for snappier UX
      setInputValue('');
      await sendMessageToThread(threadId, message);
    },
    [inputValue, status, sendMessageToThread, threadId]
  );

  const handleClearThread = useCallback(() => {
    if (messages.length === 0 || status !== 'ready') return;
    clearThreadMessages(threadId);
  }, [messages.length, status, clearThreadMessages, threadId]);

  const handleToggleChat = useCallback(() => {
    onToggleChat();
  }, [onToggleChat]);

  return (
    <div
      className={`border-subtle flex h-full w-[486px] shrink-0 flex-col border-l bg-white ${
        className ?? ''
      }`}
    >
      <div className="bg-surfaceBase flex h-full w-full flex-col">
        <ChatHeader onClearThread={handleClearThread} onToggleChat={handleToggleChat} />

        <Conversation>
          <ConversationContent>
            <div className="flex-1 space-y-4 p-3">
              {messages.map((m) => (
                <div key={m.id} className={m.role === 'user' ? 'text-right' : 'text-left'}>
                  {m.role === 'user'
                    ? m.parts.map((part, i) => {
                        if (part.type === 'text') {
                          return <UserMessage key={i} part={part} />;
                        }
                        return null;
                      })
                    : m.parts.map((part, i) => {
                        if (part.type === 'text') {
                          return <AssistantMessage key={i} part={part} />;
                        }
                        if (part.type === 'tool-call') {
                          return (
                            <ToolMessage
                              key={i}
                              part={part}
                              onSqlChange={onSqlChange}
                              runQuery={runQuery}
                            />
                          );
                        }
                        return null;
                      })}
                </div>
              ))}
              {(() => {
                const text = getLoadingMessage({
                  networkActive,
                  textStreaming,
                  textCompleted,
                  toolName: currentToolName,
                  status,
                });
                return text ? <LoadingIndicator text={text} /> : null;
              })()}
            </div>
          </ConversationContent>
        </Conversation>

        <div className="p-2">
          <ResponsivePromptInput
            value={inputValue}
            onChange={setInputValue}
            onSubmit={handleSubmit}
            disabled={status !== 'ready'}
          />
        </div>
      </div>
    </div>
  );
}
