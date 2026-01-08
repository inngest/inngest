import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

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

// Fun technical phrases that rotate while the agent is working
const LOADING_PHRASES = [
  'Analyzing schema…',
  'Indexing events…',
  'Parsing metadata…',
  'Optimizing joins…',
  'Compiling filters…',
  'Validating syntax…',
  'Mapping relations…',
  'Resolving types…',
  'Scanning indexes…',
  'Building AST…',
  'Inferring constraints…',
  'Normalizing data…',
  'Evaluating predicates…',
  'Projecting columns…',
  'Aggregating results…',
  'Planning execution…',
  'Allocating buffers…',
  'Streaming rows…',
  'Caching metadata…',
  'Rewriting queries…',
  'Reticulating splines…',
];

// Hook to rotate through loading phrases every 3 seconds
function useRotatingLoadingMessage(isLoading: boolean): string {
  const [currentIndex, setCurrentIndex] = useState(0);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (isLoading) {
      // Pick a random starting index
      setCurrentIndex(Math.floor(Math.random() * LOADING_PHRASES.length));

      // Rotate every 1.5 seconds
      intervalRef.current = setInterval(() => {
        setCurrentIndex((prev) => (prev + 1) % LOADING_PHRASES.length);
      }, 2500);
    } else {
      // Clean up interval when not loading
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [isLoading]);

  return LOADING_PHRASES[currentIndex];
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
  const { networkActive, textStreaming } = useMemo(
    () => getThreadFlags(agentThreadId),
    [getThreadFlags, agentThreadId],
  );

  // Determine if agent is actively working
  const isLoading = status !== 'ready' && (networkActive || textStreaming);

  // Get rotating loading message
  const rotatingMessage = useRotatingLoadingMessage(isLoading);

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

  // Show rotating message when loading, hide when done
  const loadingText = isLoading ? rotatingMessage : null;

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
