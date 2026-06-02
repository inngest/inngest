import type { Dispatch } from 'react';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { InsightsChartConfigPanel } from './InsightsChartConfigPanel';
import { InsightsChartEmptyState } from './InsightsChartEmptyState';
import { InsightsChartRenderer } from './InsightsChartRenderer';
import type { ChartConfig } from './types';
import type { ChartConfigAction } from './useChartConfig';

type InsightsChartViewProps = {
  config: ChartConfig;
  dispatch: Dispatch<ChartConfigAction>;
};

export function InsightsChartView({
  config,
  dispatch,
}: InsightsChartViewProps) {
  const { data, status } = useInsightsStateMachineContext();

  if (status !== 'success' || !data?.rows?.length) {
    return <InsightsChartEmptyState reason="no-data" />;
  }

  return (
    <div className="flex h-full min-h-0">
      {/* Main chart area */}
      <div className="min-w-0 flex-1 overflow-hidden">
        {config.xAxisColumn && config.yAxisColumn ? (
          <InsightsChartRenderer data={data} config={config} />
        ) : (
          <InsightsChartEmptyState reason="not-configured" />
        )}
      </div>

      {/* Config panel */}
      <div className="border-subtle w-[280px] shrink-0 overflow-y-auto border-l">
        <InsightsChartConfigPanel
          columns={data.columns}
          config={config}
          dispatch={dispatch}
        />
      </div>
    </div>
  );
}
