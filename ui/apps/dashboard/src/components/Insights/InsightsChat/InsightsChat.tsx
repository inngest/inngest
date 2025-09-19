'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  useAgents,
  type AgentStatus,
  type RealtimeEvent,
  type ToolResultPayload,
} from '@inngest/use-agents';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Conversation, ConversationContent } from './Conversation';
import { LoadingIndicator } from './LoadingIndicator';
import { ChatHeader } from './header/ChatHeader';
import { useEvents } from './hooks/use-events';
import { ResponsivePromptInput } from './input/InputField';
import { AssistantMessage } from './messages/AssistantMessage';
import { ToolMessage } from './messages/ToolMessage';
import { UserMessage } from './messages/UserMessage';

type GenerateSqlResult = {
  sql: string;
  title?: string;
  reasoning?: string;
};

type SelectEventsResult = {
  selected: {
    event_name: string;
    reason: string;
  }[];
  reason: string;
  totalCandidates: number;
};

// Using shared ToolResultPayload from @inngest/use-agents

// Tool manifest for typed onToolResult callback
type InsightsToolManifest = {
  generate_sql: ToolResultPayload<GenerateSqlResult>;
  select_events: ToolResultPayload<SelectEventsResult>;
};

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

// Helper: determine if an incoming event belongs to this thread
function isEventForThisThread(tid: unknown, threadId: string): boolean {
  return typeof tid !== 'string' || tid === threadId;
}

export function InsightsChat({ threadId }: { threadId: string }) {
  // Read required data from the Insights state context
  const {
    query: currentSql,
    queryName: tabTitle,
    onChange: onSqlChange,
    runQuery,
  } = useInsightsStateMachineContext();

  // State for the chat's input value
  const [inputValue, setInputValue] = useState('');

  // Events API hook
  const { schemas, eventTypes } = useEvents();

  // Local loading flags driven by onEvent to show “Thinking…” correctly pre-stream
  const [networkActive, setNetworkActive] = useState(false);
  const [textStreaming, setTextStreaming] = useState(false);
  const [textCompleted, setTextCompleted] = useState(false);
  const [currentToolName, setCurrentToolName] = useState<string | null>(null);

  const onEvent = useCallback(
    (evt: RealtimeEvent) => {
      try {
        switch (evt.event) {
          case 'run.started': {
            const tid = evt.data.threadId;
            if (!isEventForThisThread(tid, threadId)) return;
            setNetworkActive(true);
            setTextCompleted(false);
            setCurrentToolName(null);
            break;
          }
          case 'text.delta': {
            const tid = evt.data.threadId;
            if (!isEventForThisThread(tid, threadId)) return;
            setTextStreaming(true);
            setTextCompleted(false);
            break;
          }
          case 'tool_call.arguments.delta': {
            const tid = evt.data.threadId;
            if (!isEventForThisThread(tid, threadId)) return;
            const toolName = evt.data.toolName;
            if (typeof toolName === 'string' && toolName.length > 0) {
              setCurrentToolName(toolName);
            }
            break;
          }
          case 'part.completed': {
            const tid = evt.data.threadId;
            if (!isEventForThisThread(tid, threadId)) return;
            const { type } = evt.data;
            if (type === 'text') {
              setTextStreaming(false);
              setTextCompleted(true);
            } else if (type === 'tool-output' || type === 'tool-call') {
              setCurrentToolName(null);
            }
            break;
          }
          case 'stream.ended': {
            const tid = evt.data.threadId;
            if (!isEventForThisThread(tid, threadId)) return;
            setNetworkActive(false);
            setTextStreaming(false);
            setTextCompleted(true);
            setCurrentToolName(null);
            break;
          }
          default:
            break;
        }
      } catch {}
    },
    [threadId]
  );

  const {
    messages,
    status,
    currentThreadId,
    setCurrentThreadId,
    clearThreadMessages,
    sendMessageToThread,
  } = useAgents<InsightsToolManifest>({
    enableThreadValidation: false,
    state: () => ({
      sqlQuery: currentSql,
      eventTypes,
      schemas,
      currentQuery: currentSql,
      tabTitle,
      mode: 'insights_sql_playground',
      timestamp: Date.now(),
    }),
    onEvent,
    onToolResult: (res) => {
      try {
        if (res.toolName === 'generate_sql') {
          const sql = res.output.data.sql;
          if (sql) {
            onSqlChange(sql.trim());
            runQuery();
          }
        }
      } catch {}
    },
  });

  // Keep per-tab thread isolated and stable
  useEffect(() => {
    if (currentThreadId !== threadId) setCurrentThreadId(threadId);
  }, [currentThreadId, setCurrentThreadId, threadId]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const message = inputValue.trim();
      if (!message || status !== 'ready') return;
      // Clear input immediately for snappier UX
      setInputValue('');
      await sendMessageToThread(threadId, message);
    },
    [inputValue, status, sendMessageToThread, threadId, currentSql, eventTypes, schemas, tabTitle]
  );

  const handleClearThread = useCallback(() => {
    if (messages.length === 0 || status !== 'ready') return;
    clearThreadMessages(threadId);
  }, [messages.length, status, clearThreadMessages, threadId]);

  const handleToggleChat = useCallback(() => {
    // TODO: replace this with proper state mgmt
    return;
  }, []);

  return (
    <div className="border-subtle flex h-full w-[412px] flex-col border-l bg-white">
      <div className="bg-surfaceBase flex h-full w-full flex-col">
        <ChatHeader onClearThread={handleClearThread} onToggleChat={handleToggleChat} />

        <Conversation>
          <ConversationContent>
            <div className="flex-1 space-y-4 p-3">
              {messages.map((m) => (
                <div key={m.id} className={m.role === 'user' ? 'text-right' : 'text-left'}>
                  {m.role === 'user'
                    ? m.parts.map((p, i) => {
                        if (p.type === 'text') {
                          return <UserMessage key={i} part={p} />;
                        }
                        return null;
                      })
                    : m.parts.map((p, i) => {
                        if (p.type === 'text') {
                          return <AssistantMessage key={i} part={p} />;
                        }
                        if (p.type === 'tool-call' && p.toolName === 'generate_sql') {
                          return (
                            <ToolMessage
                              key={i}
                              part={p}
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
