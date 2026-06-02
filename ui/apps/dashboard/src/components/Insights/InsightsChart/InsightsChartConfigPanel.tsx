import { type Dispatch } from 'react';
import { LabeledCheckbox } from '@inngest/components/Checkbox/Checkbox';
import { Select, type Option } from '@inngest/components/Select/Select';
import SegmentedControl from '@inngest/components/SegmentedControl/SegmentedControl';
import { Switch, SwitchLabel, SwitchWrapper } from '@inngest/components/Switch';
import { RiBarChartLine, RiLineChartLine } from '@remixicon/react';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import type { ChartConfigAction } from './useChartConfig';
import type { ChartConfig } from './types';

type InsightsChartConfigPanelProps = {
  columns: InsightsFetchResult['columns'];
  config: ChartConfig;
  dispatch: Dispatch<ChartConfigAction>;
};

export function InsightsChartConfigPanel({
  columns,
  config,
  dispatch,
}: InsightsChartConfigPanelProps) {
  const columnOptions: Option[] = columns.map((col) => ({
    id: col.name,
    name: col.name,
  }));

  const selectedXOption =
    columnOptions.find((o) => o.id === config.xAxisColumn) ?? null;
  const selectedYOption =
    columnOptions.find((o) => o.id === config.yAxisColumn) ?? null;

  return (
    <div className="flex flex-col gap-6 p-4">
      {/* Chart Type */}
      <div className="flex flex-col gap-2">
        <span className="text-muted text-xs font-medium uppercase tracking-wide">
          Chart Type*
        </span>
        <SegmentedControl defaultValue={config.chartType}>
          <SegmentedControl.Button
            value="line"
            icon={<RiLineChartLine />}
            onClick={() =>
              dispatch({ type: 'SET_CHART_TYPE', chartType: 'line' })
            }
          >
            Line chart
          </SegmentedControl.Button>
          <SegmentedControl.Button
            value="bar"
            icon={<RiBarChartLine />}
            onClick={() =>
              dispatch({ type: 'SET_CHART_TYPE', chartType: 'bar' })
            }
          >
            Bar chart
          </SegmentedControl.Button>
        </SegmentedControl>
      </div>

      {/* X-Axis */}
      <div className="flex flex-col gap-2">
        <span className="text-muted text-xs font-medium uppercase tracking-wide">
          X-Axis
        </span>
        <Select
          onChange={(value: Option) => {
            dispatch({ type: 'SET_X_AXIS', column: value.id });
          }}
          isLabelVisible={false}
          value={selectedXOption}
          size="small"
        >
          <Select.Button size="small">
            <span className="text-basis truncate text-sm">
              {selectedXOption?.name ?? 'Choose column'}
            </span>
          </Select.Button>
          <Select.Options>
            {columnOptions.map((option) => (
              <Select.Option key={option.id} option={option}>
                {option.name}
              </Select.Option>
            ))}
          </Select.Options>
        </Select>
        <LabeledCheckbox
          id="convert-x-float"
          label={
            <span className="text-xs">
              Convert{' '}
              <code className="bg-canvasSubtle rounded px-1 text-xs">
                string
              </code>{' '}
              values to{' '}
              <code className="bg-canvasSubtle rounded px-1 text-xs">
                float
              </code>
            </span>
          }
          checked={config.convertXToFloat}
          onCheckedChange={(checked) =>
            dispatch({
              type: 'SET_CONVERT_X_TO_FLOAT',
              value: checked === true,
            })
          }
        />
      </div>

      {/* Y-Axis */}
      <div className="flex flex-col gap-2">
        <span className="text-muted text-xs font-medium uppercase tracking-wide">
          Y-Axis
        </span>
        <Select
          onChange={(value: Option) => {
            dispatch({ type: 'SET_Y_AXIS', column: value.id });
          }}
          isLabelVisible={false}
          value={selectedYOption}
          size="small"
        >
          <Select.Button size="small">
            <span className="text-basis truncate text-sm">
              {selectedYOption?.name ?? 'Choose value'}
            </span>
          </Select.Button>
          <Select.Options>
            {columnOptions.map((option) => (
              <Select.Option key={option.id} option={option}>
                {option.name}
              </Select.Option>
            ))}
          </Select.Options>
        </Select>
        <LabeledCheckbox
          id="convert-y-float"
          label={
            <span className="text-xs">
              Convert{' '}
              <code className="bg-canvasSubtle rounded px-1 text-xs">
                string
              </code>{' '}
              values to{' '}
              <code className="bg-canvasSubtle rounded px-1 text-xs">
                float
              </code>
            </span>
          }
          checked={config.convertYToFloat}
          onCheckedChange={(checked) =>
            dispatch({
              type: 'SET_CONVERT_Y_TO_FLOAT',
              value: checked === true,
            })
          }
        />
      </div>

      {/* Other options */}
      <div className="flex flex-col gap-3">
        <span className="text-muted text-xs font-medium uppercase tracking-wide">
          Other
        </span>
        <SwitchWrapper>
          <Switch
            id="show-tooltips"
            checked={config.showTooltips}
            onCheckedChange={(checked) =>
              dispatch({ type: 'SET_SHOW_TOOLTIPS', value: checked })
            }
          />
          <SwitchLabel htmlFor="show-tooltips" className="text-sm">
            Show tooltips
          </SwitchLabel>
        </SwitchWrapper>
        <SwitchWrapper>
          <Switch
            id="show-labels"
            checked={config.showLabels}
            onCheckedChange={(checked) =>
              dispatch({ type: 'SET_SHOW_LABELS', value: checked })
            }
          />
          <SwitchLabel htmlFor="show-labels" className="text-sm">
            Show labels
          </SwitchLabel>
        </SwitchWrapper>
      </div>
    </div>
  );
}
