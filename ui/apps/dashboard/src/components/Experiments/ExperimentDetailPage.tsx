import { useCallback, useMemo, useState } from 'react';

import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { Header } from '@inngest/components/Header/Header';
import { Pill } from '@inngest/components/Pill/Pill';
import { Skeleton } from '@inngest/components/Skeleton';
import { Switch, SwitchLabel, SwitchWrapper } from '@inngest/components/Switch';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiCloseLine,
  RiEqualizerLine,
  RiFlaskLine,
  RiRefreshLine,
} from '@remixicon/react';
import type { ColumnDef } from '@tanstack/react-table';
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import { pathCreator } from '@/utils/urls';

import {
  type ExperimentDetail,
  type ExperimentInsightsRow,
  type ExperimentMetadataField,
  useExperimentDetail,
} from './useExperiments';

type PanelSection = 'info' | 'scoring';

type TableRow = {
  id: string;
  dimensions: Record<string, string>;
  runCount: number;
  failureRate: number;
  percentOfTotal: number;
};

function formatPercent(value: number): string {
  return `${Intl.NumberFormat('en-US', {
    style: 'percent',
    maximumFractionDigits: 1,
  }).format(value)}`;
}

function chartDataForField(
  field: ExperimentMetadataField,
  rows: ExperimentInsightsRow[],
) {
  const values = new Map<string, number>();

  for (const row of rows) {
    const dimension = row.dimensions.find((item) => item.key === field.key);
    const key = dimension?.value ?? 'Unknown';
    values.set(key, (values.get(key) ?? 0) + row.runCount);
  }

  return [...values.entries()]
    .map(([name, runCount]) => ({ name, runCount }))
    .sort((a, b) => b.runCount - a.runCount);
}

function toTableRows(rows: ExperimentInsightsRow[]): TableRow[] {
  return rows.map((row, index) => ({
    id: `${index}`,
    dimensions: Object.fromEntries(
      row.dimensions.map((dimension) => [dimension.key, dimension.value]),
    ),
    runCount: row.runCount,
    failureRate: row.failureRate,
    percentOfTotal: row.percentOfTotal,
  }));
}

// Metric definitions for the score summary compound bar.
// Each metric becomes a stacked segment in the horizontal bar per variant.
const SCORE_METRICS = [
  {
    key: 'runCount',
    label: 'Runs',
    color: 'rgb(var(--color-primary-subtle) / 1)',
  },
  {
    key: 'failureRate',
    label: 'Failure rate',
    color: 'rgb(var(--color-tertiary-subtle) / 1)',
  },
  {
    key: 'percentOfTotal',
    label: '% of total',
    color: 'rgb(var(--color-secondary-subtle) / 1)',
  },
] as const;

function buildScoreSummaryData(rows: ExperimentInsightsRow[]) {
  // Each row becomes one horizontal compound bar.
  // The label is the joined dimension values.
  return rows.map((row) => {
    const label = row.dimensions.map((d) => d.value).join(' / ') || 'Unknown';
    return {
      label,
      runCount: row.runCount,
      failureRate: row.failureRate,
      percentOfTotal: row.percentOfTotal,
    };
  });
}

function ScoreSummaryChart({ rows }: { rows: ExperimentInsightsRow[] }) {
  const chartData = useMemo(() => buildScoreSummaryData(rows), [rows]);

  if (chartData.length === 0) {
    return (
      <div className="text-muted flex h-full items-center justify-center text-sm">
        No score data available.
      </div>
    );
  }

  const chartHeight = Math.max(200, chartData.length * 52 + 60);

  return (
    <ResponsiveContainer width="100%" height={chartHeight}>
      <BarChart
        data={chartData}
        layout="vertical"
        margin={{ top: 8, right: 24, left: 8, bottom: 8 }}
      >
        <CartesianGrid
          strokeDasharray="0"
          horizontal={false}
          className="stroke-disabled"
        />
        <XAxis
          type="number"
          tickLine={false}
          axisLine={false}
          fontSize={12}
          className="fill-muted"
        />
        <YAxis
          type="category"
          dataKey="label"
          tickLine={false}
          axisLine={false}
          fontSize={12}
          className="fill-muted"
          width={140}
        />
        <Tooltip
          cursor={false}
          wrapperStyle={{ outline: 'none' }}
          content={({ active, payload, label }) => {
            if (!active || !payload?.length) {
              return null;
            }

            return (
              <div className="bg-canvasBase shadow-tooltip rounded-md px-3 pb-2 pt-1 text-sm shadow-md">
                <div className="text-muted pb-2">{String(label)}</div>
                {payload.map((entry) => (
                  <div
                    key={String(entry.name)}
                    className="flex items-center gap-2 py-0.5"
                  >
                    <span
                      className="inline-flex h-2.5 w-2.5 rounded-sm"
                      style={{ backgroundColor: String(entry.color) }}
                    />
                    <span className="text-muted text-xs">
                      {String(entry.name)}
                    </span>
                    <span className="text-basis text-sm font-medium">
                      {typeof entry.value === 'number' && entry.value < 1
                        ? formatPercent(entry.value)
                        : Number(entry.value ?? 0).toLocaleString()}
                    </span>
                  </div>
                ))}
              </div>
            );
          }}
        />
        {SCORE_METRICS.map((metric) => (
          <Bar
            key={metric.key}
            dataKey={metric.key}
            name={metric.label}
            stackId="score"
            fill={metric.color}
            radius={[0, 0, 0, 0]}
          />
        ))}
      </BarChart>
    </ResponsiveContainer>
  );
}

function PanelIconStrip({
  activeSection,
  onSelect,
}: {
  activeSection: PanelSection | null;
  onSelect: (section: PanelSection) => void;
}) {
  const items: {
    section: PanelSection;
    icon: React.ReactNode;
    title: string;
  }[] = [
    {
      section: 'info',
      icon: <RiFlaskLine size={18} />,
      title: 'Info',
    },
    {
      section: 'scoring',
      icon: <RiEqualizerLine size={18} />,
      title: 'Scoring formula',
    },
  ];

  return (
    <div className="border-subtle flex h-full w-[56px] flex-col items-center gap-2 border-l px-3 py-2">
      {items.map((item) => (
        <button
          key={item.section}
          aria-label={item.title}
          title={item.title}
          className={cn(
            'flex h-8 w-8 items-center justify-center rounded-md transition-colors',
            activeSection === item.section
              ? 'bg-secondary-4xSubtle hover:bg-secondary-3xSubtle text-info'
              : 'text-subtle hover:bg-canvasSubtle',
          )}
          onClick={() => onSelect(item.section)}
          type="button"
        >
          {item.icon}
        </button>
      ))}
    </div>
  );
}

function InfoPanel({
  data,
  isLoading,
  experimentName,
}: {
  data: ExperimentDetail | undefined;
  isLoading: boolean;
  experimentName: string;
}) {
  return (
    <div className="flex flex-col overflow-y-auto">
      <div className="inline-flex flex-col items-start justify-start px-4 pb-6 pt-4">
        <h2 className="text-light pb-3 text-xs uppercase leading-4 tracking-wider">
          Overview
        </h2>
        <div className="flex flex-col space-y-6 self-stretch">
          <div>
            <h3 className="text-basis mb-1 flex text-sm">Experiment</h3>
            <div className="border-subtle overflow-hidden rounded border-[0.5px]">
              <table className="w-full table-fixed">
                <thead>
                  <tr className="bg-disabled border-subtle h-8 border-b-[0.5px]">
                    <td className="text-basis px-2 text-sm" colSpan={2}>
                      Details
                    </td>
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-subtle h-8 border-b-[0.5px]">
                    <td className="text-muted px-2 text-sm">Name</td>
                    <td className="text-basis px-2 text-right text-sm">
                      {isLoading ? (
                        <Skeleton className="ml-auto h-4 w-24" />
                      ) : (
                        <span className="truncate" title={experimentName}>
                          {experimentName}
                        </span>
                      )}
                    </td>
                  </tr>
                  <tr className="border-subtle h-8 border-b-[0.5px]">
                    <td className="text-muted px-2 text-sm">Type</td>
                    <td className="text-basis px-2 text-right text-sm">
                      {isLoading ? (
                        <Skeleton className="ml-auto h-4 w-20" />
                      ) : (
                        <code className="font-mono text-xs">
                          {data?.summary.selectionStrategy ?? '-'}
                        </code>
                      )}
                    </td>
                  </tr>
                  <tr className="border-subtle h-8 border-b-[0.5px]">
                    <td className="text-muted px-2 text-sm">Total runs</td>
                    <td className="text-basis px-2 text-right text-sm tabular-nums">
                      {isLoading ? (
                        <Skeleton className="ml-auto h-4 w-12" />
                      ) : (
                        data?.summary.totalRuns.toLocaleString()
                      )}
                    </td>
                  </tr>
                  <tr className="border-subtle h-8 border-b-[0.5px] last:border-b-0">
                    <td className="text-muted px-2 text-sm">Last seen</td>
                    <td className="text-basis px-2 text-right text-sm">
                      {isLoading ? (
                        <Skeleton className="ml-auto h-4 w-16" />
                      ) : data?.summary.lastSeen ? (
                        <Time format="relative" value={data.summary.lastSeen} />
                      ) : (
                        '-'
                      )}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <div>
            <h3 className="text-basis mb-1 flex text-sm">Variants</h3>
            {isLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-7 w-full" />
                <Skeleton className="h-7 w-3/4" />
              </div>
            ) : data?.summary.variants && data.summary.variants.length > 0 ? (
              <div className="flex flex-wrap gap-1.5">
                {data.summary.variants.map((variant) => (
                  <Pill key={variant} appearance="outlined">
                    {variant}
                  </Pill>
                ))}
              </div>
            ) : (
              <p className="text-muted text-sm">No variants observed.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function ScoringPanel({
  availableFields,
  selectedFieldKeys,
  onToggleField,
  isLoading,
}: {
  availableFields: ExperimentMetadataField[];
  selectedFieldKeys: string[];
  onToggleField: (key: string, enabled: boolean) => void;
  isLoading: boolean;
}) {
  return (
    <div className="flex flex-col overflow-y-auto">
      <div className="inline-flex flex-col items-start justify-start px-4 pb-6 pt-4">
        <h2 className="text-light pb-3 text-xs uppercase leading-4 tracking-wider">
          Scoring formula
        </h2>
        <div className="flex flex-col space-y-6 self-stretch">
          <div>
            <h3 className="text-basis mb-1 flex text-sm">Metadata fields</h3>
            {isLoading ? (
              <div className="flex flex-col gap-4 pt-2">
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-full" />
              </div>
            ) : availableFields.length > 0 ? (
              <div className="flex flex-col gap-4 pt-2">
                {availableFields.map((field) => {
                  const checked = selectedFieldKeys.includes(field.key);

                  return (
                    <SwitchWrapper key={field.key}>
                      <Switch
                        id={`field-${field.key}`}
                        checked={checked}
                        onCheckedChange={(nextChecked) => {
                          onToggleField(field.key, !!nextChecked);
                        }}
                      />
                      <div className="flex flex-col">
                        <SwitchLabel
                          htmlFor={`field-${field.key}`}
                          className="text-sm"
                        >
                          {field.label}
                        </SwitchLabel>
                        <span className="text-muted text-xs">
                          {field.valueType.toLowerCase()}
                        </span>
                      </div>
                    </SwitchWrapper>
                  );
                })}
              </div>
            ) : (
              <p className="text-muted pt-2 text-sm">
                No selectable metadata fields were observed for this experiment.
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function HelperPanel({
  activeSection,
  data,
  isLoading,
  experimentName,
  selectedFieldKeys,
  onToggleField,
  onClose,
}: {
  activeSection: PanelSection;
  data: ExperimentDetail | undefined;
  isLoading: boolean;
  experimentName: string;
  selectedFieldKeys: string[];
  onToggleField: (key: string, enabled: boolean) => void;
  onClose: () => void;
}) {
  const title = activeSection === 'info' ? 'Info' : 'Scoring formula';
  const icon =
    activeSection === 'info' ? (
      <RiFlaskLine className="text-subtle h-4 w-4" />
    ) : (
      <RiEqualizerLine className="text-subtle h-4 w-4" />
    );

  return (
    <div className="border-subtle flex h-full w-[280px] shrink-0 flex-col border-l">
      <div className="border-subtle flex h-[49px] shrink-0 items-center justify-between border-b px-3">
        <div className="flex items-center gap-2">
          {icon}
          <span className="text-sm font-normal">{title}</span>
        </div>
        <button
          aria-label="Close panel"
          className="hover:bg-canvasSubtle hover:text-basis text-subtle -mr-1 flex h-8 w-8 items-center justify-center rounded-md transition-colors"
          onClick={onClose}
          type="button"
        >
          <RiCloseLine size={18} />
        </button>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto">
        {activeSection === 'info' ? (
          <InfoPanel
            data={data}
            isLoading={isLoading}
            experimentName={experimentName}
          />
        ) : (
          <ScoringPanel
            availableFields={data?.availableFields ?? []}
            selectedFieldKeys={selectedFieldKeys}
            onToggleField={onToggleField}
            isLoading={isLoading}
          />
        )}
      </div>
    </div>
  );
}

export default function ExperimentDetailPage({
  environmentSlug,
  experimentName,
  selectedFieldKeys,
  onSelectedFieldKeysChange,
}: {
  environmentSlug: string;
  experimentName: string;
  selectedFieldKeys: string[];
  onSelectedFieldKeysChange: (fields: string[]) => void;
}) {
  const [activePanel, setActivePanel] = useState<PanelSection | null>('info');

  const { data, isPending, error, refetch } = useExperimentDetail({
    experimentName,
    fields: selectedFieldKeys,
  });

  const tableRows = useMemo(() => toTableRows(data?.rows ?? []), [data?.rows]);

  const tableColumns = useMemo<ColumnDef<TableRow>[]>(() => {
    const dimensionColumns = (data?.selectedFields ?? []).map<
      ColumnDef<TableRow>
    >((field) => ({
      id: field.key,
      header: field.label,
      accessorFn: (row) => row.dimensions[field.key] ?? 'Unknown',
      cell: ({ getValue }) => (
        <span className="text-basis text-sm">
          {String(getValue() ?? 'Unknown')}
        </span>
      ),
    }));

    return [
      ...dimensionColumns,
      {
        id: 'runCount',
        header: 'Runs',
        accessorFn: (row) => row.runCount,
        cell: ({ getValue }) => (
          <span className="text-basis text-sm font-medium tabular-nums">
            {Number(getValue()).toLocaleString()}
          </span>
        ),
      },
      {
        id: 'failureRate',
        header: 'Failure rate',
        accessorFn: (row) => row.failureRate,
        cell: ({ getValue }) => (
          <span className="text-muted text-sm tabular-nums">
            {formatPercent(Number(getValue()))}
          </span>
        ),
      },
      {
        id: 'percentOfTotal',
        header: '% of total',
        accessorFn: (row) => row.percentOfTotal,
        cell: ({ getValue }) => (
          <span className="text-muted text-sm tabular-nums">
            {formatPercent(Number(getValue()))}
          </span>
        ),
      },
    ];
  }, [data?.selectedFields]);

  const handleToggleField = useCallback(
    (key: string, enabled: boolean) => {
      const nextFields = enabled
        ? [...selectedFieldKeys, key]
        : selectedFieldKeys.filter((k) => k !== key);
      onSelectedFieldKeysChange(nextFields);
    },
    [selectedFieldKeys, onSelectedFieldKeysChange],
  );

  const handleSelectPanel = useCallback((section: PanelSection) => {
    setActivePanel((prev) => (prev === section ? null : section));
  }, []);

  if (error) {
    return <ErrorCard error={error} reset={() => void refetch()} />;
  }

  const isLoading = isPending && !data;
  const hasSelectedFields = data && data.selectedFields.length > 0;

  return (
    <>
      <Header
        breadcrumb={[
          {
            text: 'Experiments',
            href: pathCreator.experiments({ envSlug: environmentSlug }),
          },
          { text: experimentName },
        ]}
        action={
          <Button
            appearance="outlined"
            label="Refresh"
            icon={<RiRefreshLine />}
            iconSide="left"
            onClick={() => {
              void refetch();
            }}
          />
        }
      />

      <div className="flex min-h-0 flex-1">
        <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-3">
          {hasSelectedFields ? (
            <>
              <Card>
                <Card.Header>
                  <div className="flex items-center justify-between">
                    <div className="flex flex-col gap-0.5">
                      <span className="font-medium">Score summary</span>
                      <span className="text-muted text-xs">
                        Metrics per dimension value across selected fields.
                      </span>
                    </div>
                    <div className="flex gap-4">
                      {SCORE_METRICS.map((metric) => (
                        <span
                          key={metric.key}
                          className="inline-flex items-center text-xs"
                        >
                          <span
                            className="mr-1.5 inline-flex h-2.5 w-2.5 rounded-sm"
                            style={{ backgroundColor: metric.color }}
                          />
                          <span className="text-muted">{metric.label}</span>
                        </span>
                      ))}
                    </div>
                  </div>
                </Card.Header>
                <Card.Content>
                  <ScoreSummaryChart rows={data.rows} />
                </Card.Content>
              </Card>

              <div className="grid gap-4 xl:grid-cols-2">
                {data.selectedFields.map((field) => {
                  const chartData = chartDataForField(field, data.rows);

                  return (
                    <Card key={field.key}>
                      <Card.Header>
                        <span className="font-medium">{field.label}</span>
                        <span className="text-muted text-xs">
                          Runs grouped by {field.label.toLowerCase()}.
                        </span>
                      </Card.Header>
                      <Card.Content className="h-[280px]">
                        {chartData.length > 0 ? (
                          <ResponsiveContainer width="100%" height="100%">
                            <BarChart
                              data={chartData}
                              margin={{
                                top: 8,
                                right: 8,
                                left: 0,
                                bottom: 8,
                              }}
                            >
                              <CartesianGrid
                                strokeDasharray="0"
                                vertical={false}
                                className="stroke-disabled"
                              />
                              <XAxis
                                dataKey="name"
                                tickLine={false}
                                axisLine={false}
                                fontSize={12}
                                className="fill-muted"
                              />
                              <YAxis
                                tickLine={false}
                                axisLine={false}
                                allowDecimals={false}
                                fontSize={12}
                                className="fill-muted"
                                width={36}
                              />
                              <Tooltip
                                cursor={false}
                                wrapperStyle={{ outline: 'none' }}
                                content={({ active, payload, label }) => {
                                  if (!active || !payload?.length) {
                                    return null;
                                  }

                                  return (
                                    <div className="bg-canvasBase shadow-tooltip rounded-md px-3 pb-2 pt-1 text-sm shadow-md">
                                      <div className="text-muted pb-2">
                                        {String(label)}
                                      </div>
                                      <div className="text-basis text-sm font-medium">
                                        {Number(
                                          payload[0]?.value ?? 0,
                                        ).toLocaleString()}{' '}
                                        runs
                                      </div>
                                    </div>
                                  );
                                }}
                              />
                              <Bar
                                dataKey="runCount"
                                fill="rgb(var(--color-primary-subtle) / 1)"
                                radius={[4, 4, 0, 0]}
                              />
                            </BarChart>
                          </ResponsiveContainer>
                        ) : (
                          <div className="text-muted flex h-full items-center justify-center text-sm">
                            No data available for this field.
                          </div>
                        )}
                      </Card.Content>
                    </Card>
                  );
                })}
              </div>

              <Card>
                <Card.Header>
                  <span className="font-medium">Combined results</span>
                  <span className="text-muted text-xs">
                    All selected metadata dimensions grouped into a single
                    result set.
                  </span>
                </Card.Header>
                <Card.Content className="p-0">
                  <Table
                    columns={tableColumns}
                    data={tableRows}
                    isLoading={isPending}
                    blankState={
                      <div className="text-muted py-12 text-center text-sm">
                        No grouped results are available for the selected
                        fields.
                      </div>
                    }
                  />
                </Card.Content>
              </Card>
            </>
          ) : (
            <Card>
              <Card.Content className="text-muted flex flex-col items-center gap-2 py-12 text-center text-sm">
                <RiEqualizerLine className="text-disabled h-8 w-8" />
                <span>
                  Toggle metadata fields in the scoring formula panel to render
                  comparison charts and tables.
                </span>
              </Card.Content>
            </Card>
          )}
        </div>

        {activePanel !== null ? (
          <HelperPanel
            activeSection={activePanel}
            data={data}
            isLoading={isLoading}
            experimentName={experimentName}
            selectedFieldKeys={selectedFieldKeys}
            onToggleField={handleToggleField}
            onClose={() => setActivePanel(null)}
          />
        ) : null}

        <PanelIconStrip
          activeSection={activePanel}
          onSelect={handleSelectPanel}
        />
      </div>
    </>
  );
}
