import { useEffect, useMemo, useState } from 'react';

import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';
import { LabeledCheckbox } from '@inngest/components/Checkbox/Checkbox';
import { ErrorCard } from '@inngest/components/Error/ErrorCard';
import { Header } from '@inngest/components/Header/Header';
import { Skeleton } from '@inngest/components/Skeleton';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { RiRefreshLine } from '@remixicon/react';
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
  type ExperimentInsightsRow,
  type ExperimentMetadataField,
  useExperimentDetail,
} from './useExperiments';

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

function SummaryCard({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <Card>
      <Card.Content className="flex flex-col gap-1">
        <span className="text-muted text-xs font-medium uppercase tracking-wide">
          {label}
        </span>
        <div className="text-basis text-2xl font-semibold">{children}</div>
      </Card.Content>
    </Card>
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
  const [draftFieldKeys, setDraftFieldKeys] =
    useState<string[]>(selectedFieldKeys);

  useEffect(() => {
    setDraftFieldKeys(selectedFieldKeys);
  }, [selectedFieldKeys]);

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

  if (error) {
    return <ErrorCard error={error} reset={() => void refetch()} />;
  }

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

      <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-3">
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <SummaryCard label="Total runs">
            {isPending && !data ? (
              <Skeleton className="h-8 w-24" />
            ) : (
              data?.summary.totalRuns.toLocaleString()
            )}
          </SummaryCard>
          <SummaryCard label="Selection strategy">
            {isPending && !data ? (
              <Skeleton className="h-8 w-32" />
            ) : (
              <span className="font-mono text-lg">
                experiment.{data?.summary.selectionStrategy ?? '-'}
              </span>
            )}
          </SummaryCard>
          <SummaryCard label="Variants">
            {isPending && !data ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              data?.summary.variantCount
            )}
          </SummaryCard>
          <SummaryCard label="Last seen">
            {isPending && !data ? (
              <Skeleton className="h-8 w-28" />
            ) : data?.summary.lastSeen ? (
              <Time format="relative" value={data.summary.lastSeen} />
            ) : (
              'Unknown'
            )}
          </SummaryCard>
        </div>

        <Card>
          <Card.Header>
            <span className="font-medium">Metadata fields</span>
            <span className="text-muted text-xs">
              Select one or more observed experiment metadata fields to compare.
            </span>
          </Card.Header>
          <Card.Content className="flex flex-col gap-3">
            {isPending && !data ? (
              <div className="grid gap-3 md:grid-cols-2">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : data && data.availableFields.length > 0 ? (
              <div className="grid gap-3 md:grid-cols-2">
                {data.availableFields.map((field) => {
                  const checked = draftFieldKeys.includes(field.key);

                  return (
                    <div
                      key={field.key}
                      className="border-subtle bg-canvasSubtle rounded-md border px-4 py-3"
                    >
                      <LabeledCheckbox
                        id={field.key}
                        checked={checked}
                        onCheckedChange={(nextChecked) => {
                          const nextFields = nextChecked
                            ? [...draftFieldKeys, field.key]
                            : draftFieldKeys.filter((key) => key !== field.key);

                          setDraftFieldKeys(nextFields);
                        }}
                        label={field.label}
                        description={`Observed ${field.valueType.toLowerCase()} field`}
                      />
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="text-muted text-sm">
                No selectable experiment metadata fields were observed for this
                experiment.
              </div>
            )}
            <div className="flex items-center gap-3">
              <Button
                label="Apply fields"
                onClick={() => onSelectedFieldKeysChange(draftFieldKeys)}
                disabled={
                  draftFieldKeys.length === selectedFieldKeys.length &&
                  draftFieldKeys.every(
                    (field, index) => field === selectedFieldKeys[index],
                  )
                }
              />
              {draftFieldKeys.length > 0 ? (
                <Button
                  appearance="outlined"
                  label="Clear"
                  onClick={() => {
                    setDraftFieldKeys([]);
                    onSelectedFieldKeysChange([]);
                  }}
                />
              ) : null}
            </div>
          </Card.Content>
        </Card>

        {data && data.selectedFields.length > 0 ? (
          <>
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
                            margin={{ top: 8, right: 8, left: 0, bottom: 8 }}
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
                  All selected metadata dimensions grouped into a single result
                  set.
                </span>
              </Card.Header>
              <Card.Content className="p-0">
                <Table
                  columns={tableColumns}
                  data={tableRows}
                  isLoading={isPending}
                  blankState={
                    <div className="text-muted py-12 text-center text-sm">
                      No grouped results are available for the selected fields.
                    </div>
                  }
                />
              </Card.Content>
            </Card>
          </>
        ) : (
          <Card>
            <Card.Content className="text-muted py-12 text-center text-sm">
              Select at least one metadata field to render the comparison charts
              and table.
            </Card.Content>
          </Card>
        )}
      </div>
    </>
  );
}
