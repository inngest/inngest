'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { type AgentStatus } from '@inngest/use-agent';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Conversation, ConversationContent } from './Conversation';
import { EmptyState } from './EmptyState';
import { useInsightsChatProvider } from './InsightsChatProvider';
import { LoadingIndicator } from './LoadingIndicator';
import { ChatHeader } from './header/ChatHeader';
import { ResponsivePromptInput } from './input/InputField';
import { AssistantMessage } from './messages/AssistantMessage';
import { ToolMessage } from './messages/ToolMessage';
import { UserMessage } from './messages/UserMessage';

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
  agentThreadId,
  onToggleChat,
  className,
}: {
  agentThreadId: string;
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
    eventTypes,
    schemas,
  } = useInsightsChatProvider();

  // Derive loading flags for this thread from provider
  const { networkActive, textStreaming, textCompleted, currentToolName } = useMemo(
    () => getThreadFlags(agentThreadId),
    [getThreadFlags, agentThreadId]
  );

  // Keep per-tab thread isolated and stable
  useEffect(() => {
    if (currentThreadId !== agentThreadId) setCurrentThreadId(agentThreadId);
  }, [currentThreadId, setCurrentThreadId, agentThreadId]);

  // Keep provider's per-thread client state up to date
  useEffect(() => {
    setThreadClientState(agentThreadId, {
      sqlQuery: currentSql,
      eventTypes,
      schemas,
      currentQuery: currentSql,
      tabTitle,
      mode: 'insights_sql_playground',
      timestamp: Date.now(),
    });
  }, [setThreadClientState, agentThreadId, currentSql, eventTypes, schemas, tabTitle]);

  // Apply pending SQL from background tool-output when becoming active, then optionally auto-run
  const lastAppliedSqlRef = useRef<string | null>(null);
  useEffect(() => {
    // Only act for this thread when it's the active one
    if (currentThreadId !== agentThreadId) return;
    const sql = readAndClearPendingSql(agentThreadId);
    if (typeof sql === 'string' && sql.length > 0) {
      lastAppliedSqlRef.current = sql;
      onSqlChange(sql.trim());
    }
    if (popPendingAutoRun(agentThreadId)) {
      // Defer run slightly to allow onSqlChange to commit
      setTimeout(() => {
        runQuery();
      }, 0);
    }
    // Re-run when provider reports new pending SQL ingress
  }, [
    currentThreadId,
    agentThreadId,
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
      await sendMessageToThread(agentThreadId, message);
    },
    [inputValue, status, sendMessageToThread, agentThreadId]
  );

  const handleClearThread = useCallback(() => {
    if (messages.length === 0 || status !== 'ready') return;
    clearThreadMessages(agentThreadId);
  }, [messages.length, status, clearThreadMessages, agentThreadId]);

  const loadingText = getLoadingMessage({
    networkActive,
    textStreaming,
    textCompleted,
    toolName: currentToolName,
    status,
  });

  return (
    <div
      className={`border-subtle flex h-full w-[420px] shrink-0 flex-col border-l bg-white ${
        className ?? ''
      }`}
    >
      <div className="bg-surfaceBase flex h-full w-full flex-col">
        <ChatHeader onClearThread={handleClearThread} onToggleChat={onToggleChat} />

        <Conversation>
          <ConversationContent>
            {messages.length === 0 ? (
              <div className="flex-1 p-3">
                <EmptyState />
              </div>
            ) : (
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
                            if (part.toolName === 'generate_sql') {
                              return (
                                <ToolMessage
                                  key={i}
                                  part={part}
                                  onSqlChange={onSqlChange}
                                  runQuery={runQuery}
                                />
                              );
                            }
                            // Ignore other tool-call parts here
                            return null;
                          }
                          return null;
                        })}
                  </div>
                ))}
                {loadingText && <LoadingIndicator text={loadingText} />}
              </div>
            )}
          </ConversationContent>
        </Conversation>

        <div className="p-2 px-4 pb-5">
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
