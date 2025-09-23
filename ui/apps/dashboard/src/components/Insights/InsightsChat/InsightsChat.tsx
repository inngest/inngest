'use client';

import { useCallback, useEffect, useState } from 'react';
import { type AgentStatus, type RealtimeEvent } from '@inngest/use-agents';

import { useAllEventTypes } from '@/components/EventTypes/useEventTypes';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Conversation, ConversationContent } from './Conversation';
import { LoadingIndicator } from './LoadingIndicator';
import { ChatHeader } from './header/ChatHeader';
import { ResponsivePromptInput } from './input/InputField';
import { AssistantMessage } from './messages/AssistantMessage';
import { ToolMessage } from './messages/ToolMessage';
import { UserMessage } from './messages/UserMessage';
import { useInsightsAgent } from './useInsightsAgent';

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

// Helper: determine if an incoming event belongs to this thread
function isEventForThisThread(tid: unknown, threadId: string): boolean {
  return typeof tid !== 'string' || tid === threadId;
}

export function InsightsChat({
  threadId,
  onToggleChat,
}: {
  threadId: string;
  onToggleChat: () => void;
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
              break;
            }
            if (type === 'tool-output' || type === 'tool-call') {
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
  } = useInsightsAgent({
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
          const sql = res.data.sql;
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
    <div className="border-subtle flex h-full w-[486px] flex-col border-l bg-white">
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
