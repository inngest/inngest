import { useCallback, useEffect, useMemo, useState } from 'react';
import { type AgentStatus } from '@inngest/use-agent';

import { useSQLEditorActions } from '@/components/Insights/InsightsSQLEditor/SQLEditorContext';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { Conversation, ConversationContent } from './Conversation';
import { EmptyState } from './EmptyState';
import { useInsightsChatProvider } from './InsightsChatProvider';
import { LoadingIndicator } from './LoadingIndicator';
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

type InsightsChatProps = {
  agentThreadId: string;
  className?: string;
};

export function InsightsChat({ agentThreadId, className }: InsightsChatProps) {
  // Read required data from the Insights state context
  const { query: currentSql, queryName: tabTitle } =
    useInsightsStateMachineContext();

  // Get SQL editor actions service (may be null if not in query tab context)
  const editorActions = useSQLEditorActions();

  // State for the chat's input value
  const [inputValue, setInputValue] = useState('');

  // Provider-backed agent state and actions
  const {
    messages,
    status,
    currentThreadId,
    sendMessageToThread,
    getThreadFlags,
    getLatestGeneratedSql,
    latestSqlVersion,
    setThreadClientState,
    eventTypes,
    schemas,
  } = useInsightsChatProvider();

  // Derive loading flags for this thread from provider
  const { networkActive, textStreaming, textCompleted, currentToolName } =
    useMemo(
      () => getThreadFlags(agentThreadId),
      [getThreadFlags, agentThreadId],
    );

  // Thread switching is handled by ActiveThreadBridge at the TabManager level

  // Client state is captured at send-time; avoid continuous effects here

  // When active, auto-apply latest generated SQL whenever version changes
  useEffect(() => {
    if (currentThreadId !== agentThreadId) return;
    if (!editorActions) return; // Not in query tab context
    const latest = getLatestGeneratedSql(agentThreadId);
    if (!latest) return;

    // Use the SQL editor service to set query and run it
    editorActions.setQueryAndRun(latest);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    currentThreadId,
    agentThreadId,
    // getLatestGeneratedSql is stable, don't include it
    latestSqlVersion,
    // editorActions is stable, don't include it
  ]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const message = inputValue.trim();
      if (!message || status !== 'ready') return;
      // Clear input immediately for snappier UX
      setInputValue('');
      // Capture client state snapshot at send-time
      try {
        setThreadClientState(agentThreadId, {
          sqlQuery: currentSql,
          eventTypes,
          schemas,
          currentQuery: currentSql,
          tabTitle,
          mode: 'insights_sql_playground',
          timestamp: Date.now(),
        });
      } catch {}
      await sendMessageToThread(agentThreadId, message);
    },
    [
      inputValue,
      status,
      sendMessageToThread,
      agentThreadId,
      setThreadClientState,
      currentSql,
      eventTypes,
      schemas,
      tabTitle,
    ],
  );

  const loadingText = getLoadingMessage({
    networkActive,
    textStreaming,
    textCompleted,
    toolName: currentToolName,
    status,
  });

  return (
    <div
      className={`border-subtle flex h-full w-full shrink-0 flex-col border-l bg-white ${
        className ?? ''
      }`}
    >
      <div className="bg-surfaceBase flex h-full w-full flex-col">
        <Conversation>
          <ConversationContent>
            {messages.length === 0 ? (
              <div className="flex-1 p-3">
                <EmptyState />
              </div>
            ) : (
              <div className="flex-1 space-y-4 p-3">
                {messages.map((m) => (
                  <div
                    key={m.id}
                    className={m.role === 'user' ? 'text-right' : 'text-left'}
                  >
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
                              return <ToolMessage key={i} part={part} />;
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
