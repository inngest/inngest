'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useUser } from '@clerk/nextjs';
import {
  createInMemorySessionTransport,
  useAgents,
  type AgentStatus,
  type RealtimeEvent,
  type TextUIPart,
  type ToolCallUIPart,
} from '@inngest/use-agents';
import { v4 as uuidv4 } from 'uuid';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Conversation, ConversationContent } from './Conversation';
import { LoadingIndicator } from './LoadingIndicator';
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

export function InsightsChat() {
  // Read required data from the Insights state context
  const {
    query: currentSql,
    queryName: tabTitle,
    onChange: onSqlChange,
    runQuery,
  } = useInsightsStateMachineContext();

  // Generate a unique thread ID for the initial chat
  const [threadId] = useState<string>(() => uuidv4());

  // State for the chat's input value
  const [inputValue, setInputValue] = useState('');

  // Get the user from the Clerk useUser hook
  const { user } = useUser();

  // Events API hook
  const { schemas, eventTypes } = useEvents();

  // Transport: ephemeral threads, delegate network to our API
  const transport = useMemo(() => {
    // InMemorySessionTransport only delegates sendMessage and getRealtimeToken to HTTP.
    // Defaults are /api/chat and /api/realtime/token, so no config needed.
    return createInMemorySessionTransport();
  }, []);

  // Track active stream lifecycle; set true on submit, false on 'stream.ended'
  const [streamActive, setStreamActive] = useState(false);
  const [networkActive, setNetworkActive] = useState(false);
  const [textStreaming, setTextStreaming] = useState(false);
  const [textCompleted, setTextCompleted] = useState(false);
  const [currentToolName, setCurrentToolName] = useState<string | null>(null);

  // Event data is fetched inside useEvents

  const {
    messages,
    sendMessage,
    status,
    currentThreadId,
    setCurrentThreadId,
    clearThreadMessages,
  } = useAgents({
    enableThreadValidation: false,
    transport,
    userId: user?.id,
    channelKey: user?.id ? `insights:${user.id}` : undefined,
    state: () => ({
      sqlQuery: currentSql,
      // Insights network context for Event Matcher and Query Writer
      eventTypes,
      schemas,
      currentQuery: currentSql,
      tabTitle,
      mode: 'insights_sql_playground',
      timestamp: Date.now(),
    }),
    onEvent: (evt: RealtimeEvent, meta: { scope?: string }) => {
      try {
        switch (evt.event) {
          case 'run.started': {
            if (meta.scope === 'network') {
              setNetworkActive(true);
              setTextCompleted(false);
              setCurrentToolName(null);
            }
            break;
          }
          case 'stream.ended': {
            if (meta.scope === 'network') {
              setStreamActive(false);
              setNetworkActive(false);
              setTextStreaming(false);
              setTextCompleted(true);
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
            type CreatedData = { type?: string; metadata?: { toolName?: string } };
            const data = evt.data as CreatedData | undefined;
            if (data?.type === 'tool-call') {
              const tn = data.metadata?.toolName;
              setCurrentToolName((typeof tn === 'string' && tn) || null);
            }
            break;
          }
          case 'part.completed': {
            type CompletedData = { type?: string };
            const data = evt.data as CompletedData | undefined;
            if (data?.type === 'text') {
              setTextStreaming(false);
              setTextCompleted(true);
            } else if (data?.type === 'tool-output' || data?.type === 'tool-call') {
              setCurrentToolName(null);
            }
            break;
          }
          default:
            break;
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
      if (!inputValue.trim() || streamActive) return;
      setStreamActive(true);
      await sendMessage(inputValue);
      setInputValue('');
    },
    [inputValue, streamActive, sendMessage]
  );

  return (
    <div className="flex h-full w-[412px] flex-col border-l border-gray-200 bg-white">
      <div className="bg-surfaceBase flex h-full w-full flex-col">
        <div className="border-border-muted flex items-center justify-between border-b px-3 py-2">
          <div className="text-text-basis text-sm font-medium">AI Assistant</div>
          <button
            className="text-text-subtle hover:text-text-basis text-xs"
            onClick={() => clearThreadMessages(threadId)}
            disabled={messages.length === 0 || streamActive}
          >
            Clear
          </button>
        </div>

        <Conversation>
          <ConversationContent>
            <div className="flex-1 space-y-3 p-3">
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
                      if (
                        p.type === 'tool-call' &&
                        (p as ToolCallUIPart).toolName === 'generate_sql'
                      ) {
                        return (
                          <ToolMessage
                            key={i}
                            part={p as ToolCallUIPart}
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
            disabled={streamActive}
          />
        </div>
      </div>
    </div>
  );
}
