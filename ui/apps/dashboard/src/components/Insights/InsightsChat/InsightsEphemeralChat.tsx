'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useUser } from '@clerk/nextjs';
import { createInMemorySessionTransport, useAgents } from '@inngest/use-agents';
import { v4 as uuidv4 } from 'uuid';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';

type Props = {
  tabTitle: string;
  currentSql: string;
  onSqlChange: (sql: string) => void;
};

export function InsightsEphemeralChat({ tabTitle, currentSql, onSqlChange }: Props) {
  const [threadId] = useState<string>(() => uuidv4());
  const [inputValue, setInputValue] = useState('');
  const { user } = useUser();
  const { runQuery } = useInsightsStateMachineContext();

  // Transport: ephemeral threads, delegate network to our API
  const transport = useMemo(() => {
    // InMemorySessionTransport only delegates sendMessage and getRealtimeToken to HTTP.
    // Defaults are /api/chat and /api/realtime/token, so no config needed.
    return createInMemorySessionTransport();
  }, []);

  const {
    messages,
    sendMessage,
    status,
    currentThreadId,
    setCurrentThreadId,
    clearThreadMessages,
    rehydrateMessageState,
  } = useAgents({
    enableThreadValidation: false,
    transport,
    userId: user?.id,
    channelKey: user?.id ? `insights:${user.id}` : undefined,
    state: () => ({
      sqlQuery: currentSql,
      tabTitle,
      mode: 'insights_sql_playground',
      timestamp: Date.now(),
    }),
    onStateRehydrate: (messageState) => {
      if (messageState && typeof messageState === 'object') {
        const sql = (messageState as Record<string, unknown>).sqlQuery;
        if (typeof sql === 'string' && sql !== currentSql) onSqlChange(sql);
      }
    },
  });

  // Keep per-tab thread isolated and stable
  useEffect(() => {
    if (currentThreadId !== threadId) setCurrentThreadId(threadId);
  }, [currentThreadId, setCurrentThreadId, threadId]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!inputValue.trim() || status !== 'idle') return;
      await sendMessage(inputValue);
      setInputValue('');
    },
    [inputValue, status, sendMessage]
  );

  return (
    <div className="flex h-full w-full flex-col">
      <div className="flex items-center justify-between border-b border-gray-200 px-3 py-2">
        <div className="text-sm font-medium text-gray-700">AI Assistant</div>
        <button
          className="text-xs text-gray-500 hover:text-gray-700"
          onClick={() => clearThreadMessages(threadId)}
          disabled={messages.length === 0 || status !== 'idle'}
        >
          Clear
        </button>
      </div>

      <div className="flex-1 space-y-3 overflow-auto p-3">
        {messages.map((m) => (
          <div key={m.id} className={m.role === 'user' ? 'text-right' : 'text-left'}>
            <div
              className="inline-block max-w-[340px] whitespace-pre-wrap rounded-md px-3 py-2 text-sm"
              style={{
                background: m.role === 'user' ? '#F3F4F6' : '#EEF2FF',
                color: '#111827',
              }}
            >
              {m.parts.map((p, i) => {
                if (p.type === 'text') {
                  return <div key={i}>{(p as any).content}</div>;
                }
                if (p.type === 'tool-call') {
                  const tool = p as any;
                  const output = tool.output;
                  let suggestedSql: string | null = null;
                  try {
                    if (typeof output === 'string') {
                      // try parse JSON, else treat as raw sql
                      const parsed = JSON.parse(output);
                      if (parsed && typeof parsed === 'object') {
                        if (typeof (parsed as any).sql === 'string')
                          suggestedSql = (parsed as any).sql;
                        else if (typeof (parsed as any).query === 'string')
                          suggestedSql = (parsed as any).query;
                      }
                    } else if (output && typeof output === 'object') {
                      if (typeof (output as any).sql === 'string')
                        suggestedSql = (output as any).sql;
                      else if (typeof (output as any).query === 'string')
                        suggestedSql = (output as any).query;
                    }
                  } catch {
                    /* ignore */
                  }
                  return (
                    <div key={i} className="text-xs">
                      <div className="font-medium">
                        Tool: {tool.toolName || tool.metadata?.toolName}
                      </div>
                      {tool.input && (
                        <pre className="mt-1 overflow-auto rounded bg-white/70 p-2 text-[11px]">
                          {typeof tool.input === 'string'
                            ? tool.input
                            : JSON.stringify(tool.input, null, 2)}
                        </pre>
                      )}
                      {tool.output !== undefined && (
                        <pre className="mt-1 overflow-auto rounded bg-white/70 p-2 text-[11px]">
                          {typeof tool.output === 'string'
                            ? tool.output
                            : JSON.stringify(tool.output, null, 2)}
                        </pre>
                      )}
                      {suggestedSql && (
                        <div className="mt-2">
                          <button
                            type="button"
                            className="rounded border border-gray-300 bg-white px-2 py-1 text-[11px] hover:bg-gray-50"
                            onClick={() => {
                              onSqlChange(suggestedSql!);
                              try {
                                runQuery();
                              } catch {}
                            }}
                          >
                            Insert SQL & Run
                          </button>
                        </div>
                      )}
                    </div>
                  );
                }
                return null;
              })}
            </div>
          </div>
        ))}
      </div>

      <form onSubmit={handleSubmit} className="border-t border-gray-200 p-2">
        <input
          className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm outline-none focus:border-gray-400"
          placeholder="Ask anything…"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          disabled={status !== 'idle'}
        />
      </form>
    </div>
  );
}
