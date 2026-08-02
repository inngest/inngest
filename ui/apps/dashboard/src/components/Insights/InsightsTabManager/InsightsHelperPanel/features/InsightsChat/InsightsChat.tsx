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
  'Analyzing schema\u2026',
  'Indexing events\u2026',
  'Parsing metadata\u2026',
  'Optimizing joins\u2026',
  'Compiling filters\u2026',
  'Validating syntax\u2026',
  'Mapping relations\u2026',
  'Resolving types\u2026',
  'Scanning indexes\u2026',
  'Building AST\u2026',
  'Inferring constraints\u2026',
  'Normalizing data\u2026',
  'Evaluating predicates\u2026',
  'Projecting columns\u2026',
  'Aggregating results\u2026',
  'Planning execution\u2026',
  'Allocating buffers\u2026',
  'Streaming rows\u2026',
  'Caching metadata\u2026',
  'Rewriting queries\u2026',
  'Reticulating splines\u2026',
];

// Hook to rotate through loading phrases every 3 seconds
function useRotatingLoadingMessage(isLoading: boolean): string {
  const [currentIndex, setCurrentIndex] = useState(0);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }

    if (isLoading) {
      setCurrentIndex(Math.floor(Math.random() * LOADING_PHRASES.length));

      intervalRef.current = setInterval(() => {
        setCurrentIndex((prev) => (prev + 1) % LOADING_PHRASES.length);
      }, 2500);
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
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
  const {
    query: currentSql,
    queryName: tabTitle,
    data: queryData,
    error: queryError,
    status: queryStatus,
  } = useInsightsStateMachineContext();

  const editorActions = useSQLEditorActions();

  const [inputValue, setInputValue] = useState('');

  const {
    messages,
    currentThreadId,
    sendMessageToThread,
    getThreadFlags,
    getLatestGeneratedSql,
    latestSqlVersion,
    setThreadClientState,
    eventTypes,
    schemas,
  } = useInsightsChatProvider();

  const { networkActive: isLoading } = useMemo(
    () => getThreadFlags(agentThreadId),
    [getThreadFlags, agentThreadId],
  );

  const rotatingMessage = useRotatingLoadingMessage(isLoading);

  // The agent no longer validates its SQL mid-run, so the auto-apply below is
  // where a broken query surfaces. Holds the applied SQL until the run it
  // triggered reports back, at which point the result is this query's.
  const appliedSqlRef = useRef<null | { sql: string; settled: boolean }>(null);
  const autoCorrectedRef = useRef(false);

  // When active, auto-apply latest generated SQL whenever version changes
  useEffect(() => {
    if (currentThreadId !== agentThreadId) return;
    if (!editorActions) return;
    const latest = getLatestGeneratedSql(agentThreadId);
    if (!latest) return;

    appliedSqlRef.current = { sql: latest, settled: false };
    editorActions.setQueryAndRun(latest);
  }, [currentThreadId, agentThreadId, latestSqlVersion]);

  // Send one correction turn when the applied query fails. Capped at one per
  // user message, so a correction that also fails ends there.
  useEffect(() => {
    const applied = appliedSqlRef.current;
    if (!applied || applied.settled) return;
    if (queryStatus === 'initial' || queryStatus === 'loading') return;

    applied.settled = true;
    if (queryStatus !== 'error') return;
    if (autoCorrectedRef.current || isLoading) return;

    const reasons =
      queryData?.diagnostics
        .filter((d) => d.severity === 'error')
        .map((d) => `- [${d.code || 'error'}] ${d.message}`) ?? [];
    if (reasons.length === 0 && queryError) {
      reasons.push(`- ${queryError.message}`);
    }
    if (reasons.length === 0) return;

    autoCorrectedRef.current = true;
    void sendMessageToThread(
      agentThreadId,
      `That query failed to run:\n\n${applied.sql}\n\nErrors:\n${reasons.join(
        '\n',
      )}\n\nPlease fix it.`,
    );
  }, [
    queryStatus,
    queryData,
    queryError,
    isLoading,
    agentThreadId,
    sendMessageToThread,
  ]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const message = inputValue.trim();
      if (!message || isLoading) return;
      setInputValue('');
      autoCorrectedRef.current = false;
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
      isLoading,
      sendMessageToThread,
      agentThreadId,
      setThreadClientState,
      currentSql,
      eventTypes,
      schemas,
      tabTitle,
    ],
  );

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
            disabled={isLoading}
          />
        </div>
      </div>
    </div>
  );
}
