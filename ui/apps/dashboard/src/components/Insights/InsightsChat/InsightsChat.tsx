'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  useAgents,
  type AgentStatus,
  type RealtimeEvent,
  type TextUIPart,
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

export function InsightsChat({ tabId, threadId }: { tabId: string; threadId: string }) {
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

  const partIdToToolNameRef = useRef<Map<string, string>>(new Map());
  const autoRanPartIdsRef = useRef<Set<string>>(new Set());

  const onEvent = useCallback(
    (evt: RealtimeEvent) => {
      try {
        const data = ((evt as any).data || {}) as Record<string, unknown>;
        const evtThreadId = typeof data['threadId'] === 'string' ? data['threadId'] : undefined;
        if (evtThreadId && evtThreadId !== threadId) return; // ignore other threads

        switch (evt.event) {
          case 'run.started': {
            const scope = typeof data['scope'] === 'string' ? data['scope'] : undefined;
            if (scope === 'network') {
              setNetworkActive(true);
              setTextCompleted(false);
              setCurrentToolName(null);
            }
            break;
          }
          case 'text.delta': {
            setTextStreaming(true);
            setTextCompleted(false);
            break;
          }
          case 'part.created': {
            const type = typeof data['type'] === 'string' ? data['type'] : undefined;
            const tn =
              typeof (data as { metadata?: { toolName?: string } }).metadata?.toolName === 'string'
                ? (data as { metadata?: { toolName?: string } }).metadata!.toolName
                : undefined;
            const partId = typeof data['partId'] === 'string' ? data['partId'] : undefined;
            if (type === 'tool-call') setCurrentToolName(tn || null);
            if ((type === 'tool-output' || type === 'tool-call') && partId && tn) {
              partIdToToolNameRef.current.set(partId, tn);
            }
            break;
          }
          case 'part.completed': {
            const type = typeof data['type'] === 'string' ? data['type'] : undefined;
            const partId = typeof data['partId'] === 'string' ? data['partId'] : undefined;
            if (type === 'text') {
              setTextStreaming(false);
              setTextCompleted(true);
            } else if (type === 'tool-output' || type === 'tool-call') {
              setCurrentToolName(null);
              // Auto-paste and run SQL when generate_sql tool output completes
              if (type === 'tool-output' && partId && !autoRanPartIdsRef.current.has(partId)) {
                const toolName = partIdToToolNameRef.current.get(partId);
                if (toolName === 'generate_sql') {
                  autoRanPartIdsRef.current.add(partId);
                  const finalContent = (data as { finalContent?: unknown }).finalContent;
                  let sql: string | null = null;
                  try {
                    if (finalContent && typeof finalContent === 'object') {
                      const obj = finalContent as Record<string, unknown>;
                      const envelope = (obj as { data?: unknown }).data as
                        | Record<string, unknown>
                        | undefined;
                      const candidate = envelope?.sql ?? (obj as { sql?: unknown }).sql;
                      if (typeof candidate === 'string' && candidate.trim()) sql = candidate.trim();
                    } else if (typeof finalContent === 'string') {
                      try {
                        const parsed = JSON.parse(finalContent) as Record<string, unknown>;
                        const envelope = (parsed as { data?: unknown }).data as
                          | Record<string, unknown>
                          | undefined;
                        const candidate = envelope?.sql ?? (parsed as { sql?: unknown }).sql;
                        if (typeof candidate === 'string' && candidate.trim())
                          sql = candidate.trim();
                      } catch {
                        // Not JSON; ignore
                      }
                    }
                  } catch {}
                  if (sql) {
                    try {
                      onSqlChange(sql);
                      runQuery();
                    } catch {}
                  }
                }
              }
            }
            break;
          }
          case 'stream.ended': {
            const scope = typeof data['scope'] === 'string' ? data['scope'] : undefined;
            if (scope === 'network') {
              setNetworkActive(false);
              setTextStreaming(false);
              setTextCompleted(true);
              setCurrentToolName(null);
            }
            break;
          }
          default:
            break;
        }
      } catch {}
    },
    [threadId, onSqlChange, runQuery]
  );

  const {
    messages,
    status,
    currentThreadId,
    setCurrentThreadId,
    clearThreadMessages,
    sendMessageToThread,
  } = useAgents({
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
      await sendMessageToThread(threadId, message, {
        state: () => ({
          sqlQuery: currentSql,
          eventTypes,
          schemas,
          currentQuery: currentSql,
          tabTitle,
          mode: 'insights_sql_playground',
          timestamp: Date.now(),
        }),
      });
    },
    [inputValue, status, sendMessageToThread, threadId, currentSql, eventTypes, schemas, tabTitle]
  );

  const handleClearThread = useCallback(() => {
    if (messages.length === 0 || status !== 'ready') return;
    clearThreadMessages(threadId);
  }, [messages.length, status, clearThreadMessages, threadId]);

  const handleToggleChat = useCallback(() => {
    try {
      window.dispatchEvent(
        new CustomEvent('insights:toggle-chat', { detail: { tabId, threadId } })
      );
    } catch {}
  }, [tabId, threadId]);

  return (
    <div className="border-subtle flex h-full w-[412px] flex-col border-l bg-white">
      <div className="bg-surfaceBase flex h-full w-full flex-col">
        <ChatHeader onClearThread={handleClearThread} onToggleChat={handleToggleChat} />

        <Conversation>
          <ConversationContent>
            <div className="flex-1 space-y-4 p-3">
              {messages.map((m) => (
                <div key={m.id} className={m.role === 'user' ? 'text-right' : 'text-left'}>
                  {m.role === 'user' ? (
                    <UserMessage
                      message={{
                        content: m.parts
                          .filter((p) => p.type === 'text')
                          .map((p) => (p as TextUIPart).content)
                          .join(''),
                      }}
                    />
                  ) : (
                    m.parts.map((p, i) => {
                      if (p.type === 'text') {
                        return <AssistantMessage key={i} part={p} />;
                      }
                      if (p.type === 'tool-call' && (p as any).toolName === 'generate_sql') {
                        return (
                          <ToolMessage
                            key={i}
                            part={p as any}
                            onSqlChange={onSqlChange}
                            runQuery={runQuery}
                          />
                        );
                      }
                      return null;
                    })
                  )}
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
