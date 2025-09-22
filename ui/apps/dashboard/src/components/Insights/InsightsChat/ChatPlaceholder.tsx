'use client';

import { InsightsEphemeralChat } from '@/components/Insights/InsightsChat/InsightsEphemeralChat';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';

export function ChatPlaceholder() {
  const { query, queryName, onChange } = useInsightsStateMachineContext();
  return (
    <div className="flex h-full w-[412px] flex-col border-l border-gray-200 bg-white">
      <InsightsEphemeralChat tabTitle={queryName} currentSql={query} onSqlChange={onChange} />
    </div>
  );
}
