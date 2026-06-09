import { useCallback, useMemo, useState } from 'react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { useColumns as useFunctionColumns } from '@inngest/components/Functions/columns';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Table } from '@inngest/components/Table';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiArrowDownSLine,
  RiArrowRightUpLine,
  RiArrowUpSLine,
  RiCalendarLine,
  RiGroupLine,
  RiShieldCheckLine,
} from '@remixicon/react';
import { Link, useNavigate } from '@tanstack/react-router';

import { useEnvironment } from '@/components/Environments/environment-context';
import { pathCreator } from '@/utils/urls';

import {
  TIME_RANGE_OPTIONS,
  useInfraDashboardData,
} from './useInfraDashboardData';
import type {
  InfraDashboardPlaceholders,
  InfraPlan,
  InfraPlanSku,
  InfraTier,
  InfraTierId,
} from './placeholderData';
import {
  billingCycleDaysRemaining,
  formatCompactNumber,
  formatPercent,
} from './utils';

export function InfraDashboard() {
  const env = useEnvironment();
  const { data, error, fetching } = useInfraDashboardData(
    TIME_RANGE_OPTIONS[0],
  );
  const placeholders = data.placeholders;
  const [selectedPlanSku, setSelectedPlanSku] = useState<InfraPlanSku>(
    placeholders.defaultPlanSku,
  );
  const billingDays = billingCycleDaysRemaining(data.billingNextInvoiceDate);
  const selectedPlan =
    placeholders.infraPlans.find((plan) => plan.sku === selectedPlanSku) ??
    placeholders.infraPlans[0];

  return (
    <div className="bg-canvasBase flex min-h-full w-full flex-col px-4 py-4 lg:px-6">
      <header className="mb-4 flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-basis text-xl font-medium leading-7">
              {env.name}
            </h1>
          </div>
        </div>

        <div className="flex w-fit max-w-full flex-col items-stretch gap-2">
          <InfraPlanDropdown
            plans={placeholders.infraPlans}
            selectedPlan={selectedPlan}
            onSelect={setSelectedPlanSku}
          />
          <div className="text-muted flex w-full items-center justify-between gap-3 text-xs">
            <div className="flex min-w-0 items-center gap-2">
              <span className="truncate opacity-70">
                Billing cycle ends in {billingDays} days
              </span>
            </div>
            <a
              className="text-link shrink-0 hover:underline"
              href={pathCreator.billing({ tab: 'usage' })}
            >
              View usage
            </a>
          </div>
        </div>
      </header>

      {error && (
        <div className="bg-error border-error text-error mb-4 rounded border px-3 py-2 text-sm">
          Some dashboard data failed to load. Placeholder-backed sections are
          still shown.
        </div>
      )}

      <section className="mb-4 grid grid-cols-1 gap-2 md:grid-cols-3">
        <KpiCard
          fetching={fetching}
          label="Events received"
          value={formatCompactNumber(data.eventsReceived)}
        />
        <KpiCard
          fetching={fetching}
          label="Executions ran (runs + steps)"
          value={formatCompactNumber(data.executionsRan)}
        />
        <KpiCard
          fetching={fetching}
          label="Current backlog"
          value={formatCompactNumber(data.backlogDepth)}
        />
        {/*
        TODO: Restore these when traces and scores have live billing-period data.
        <KpiCard
          fetching={fetching}
          label="Traces received"
          value={formatCompactNumber(placeholders.monthlyTracesReceived)}
        />
        <KpiCard
          fetching={fetching}
          label="Scores processed"
          value={formatCompactNumber(placeholders.monthlyScoresProcessed)}
        />
        */}
      </section>

      <InfraFlowPanel
        backlogDepth={data.backlogDepth}
        currentConcurrency={data.currentConcurrency}
        fetching={fetching}
        placeholders={placeholders}
      />

      <MostRanFunctions
        envSlug={env.slug}
        rows={data.topFunctions}
        fetching={fetching}
      />
    </div>
  );
}

function KpiCard({
  className,
  delta,
  fetching,
  label,
  progress,
  value,
}: {
  className?: string;
  delta?: string;
  fetching: boolean;
  label: string;
  progress?: number;
  value: string;
}) {
  return (
    <div
      className={cn(
        'border-subtle bg-canvasBase min-h-[92px] rounded-md border p-4',
        className,
      )}
    >
      <div className="text-muted mb-1 text-sm">{label}</div>
      {fetching ? (
        <Skeleton className="h-8 w-24" />
      ) : (
        <div className="flex items-end gap-2">
          <div className="text-basis text-3xl font-medium leading-8">
            {value}
          </div>
          {delta && (
            <div className="text-primary-intense mb-1 flex items-center gap-0.5 text-xs">
              <RiArrowRightUpLine className="h-3 w-3" />
              {delta}
            </div>
          )}
        </div>
      )}
      {typeof progress === 'number' && (
        <div className="mt-3 flex items-center gap-3">
          <div className="bg-canvasMuted h-1 w-full overflow-hidden rounded-full">
            <div
              className="bg-primary-moderate h-full rounded-full"
              style={{ width: `${progress}%` }}
            />
          </div>
          <span className="text-basis text-xs">{formatPercent(progress)}</span>
        </div>
      )}
    </div>
  );
}

function InfraPlanDropdown({
  onSelect,
  plans,
  selectedPlan,
}: {
  onSelect: (sku: InfraPlanSku) => void;
  plans: InfraPlan[];
  selectedPlan: InfraPlan;
}) {
  const [open, setOpen] = useState(false);

  const selectPlan = (sku: InfraPlanSku) => {
    onSelect(sku);
    setOpen(false);
  };

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <button
          className="border-muted bg-canvasBase text-basis hover:bg-canvasSubtle focus:ring-primary-moderate flex max-w-full items-center justify-between gap-2 rounded border px-2 py-1 text-xs focus:outline-none focus:ring-2"
          type="button"
        >
          <span className="flex min-w-0 flex-wrap items-center gap-2">
            <span className="bg-canvasMuted rounded px-1.5 py-0.5 font-medium">
              {selectedPlan.sku}
            </span>
            <span>{selectedPlan.eventStream}</span>
            <span className="text-disabled">·</span>
            <span>{selectedPlan.queueDepth} depth</span>
            <span className="text-disabled">·</span>
            <span>{selectedPlan.execConcurrency} concurrency</span>
          </span>
          {open ? (
            <RiArrowUpSLine className="h-3.5 w-3.5 shrink-0" />
          ) : (
            <RiArrowDownSLine className="h-3.5 w-3.5 shrink-0" />
          )}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-[min(calc(100vw-2rem),640px)] overflow-hidden p-0"
      >
        <div className="border-subtle bg-canvasSubtle flex items-center gap-1 border-b p-2">
          <button
            className="border-subtle text-muted inline-flex items-center gap-1.5 rounded border px-2 py-1 text-xs"
            disabled
            type="button"
          >
            <RiShieldCheckLine className="h-3.5 w-3.5" />
            Single-tenant
          </button>
          <button
            className="bg-canvasBase border-muted text-basis inline-flex items-center gap-1.5 rounded border px-2 py-1 text-xs font-medium"
            type="button"
          >
            <RiGroupLine className="h-3.5 w-3.5" />
            Shared
          </button>
        </div>

        <div className="overflow-x-auto">
          <div className="min-w-[580px]">
            <div className="bg-canvasSubtle text-muted grid grid-cols-[96px_140px_140px_160px_1fr] px-3 py-2 text-left text-[11px] font-medium uppercase">
              <span>SKU</span>
              <span>Event stream</span>
              <span>Queue depth</span>
              <span>Exec concurrency</span>
              <span className="text-right">Price / mo</span>
            </div>
            {plans.map((plan) => {
              const isSelected = plan.sku === selectedPlan.sku;

              return (
                <button
                  className={cn(
                    'border-subtle text-basis hover:bg-canvasSubtle grid w-full grid-cols-[96px_140px_140px_160px_1fr] items-center border-t px-3 py-2.5 text-left text-xs',
                    isSelected && 'bg-canvasSubtle',
                  )}
                  key={plan.sku}
                  onClick={() => selectPlan(plan.sku)}
                  type="button"
                >
                  <span>
                    <span className="border-muted bg-canvasBase inline-flex rounded border px-1.5 py-0.5 font-medium">
                      {plan.sku}
                    </span>
                  </span>
                  <PlanMetric value={plan.eventStream} />
                  <PlanMetric value={plan.queueDepth} />
                  <PlanMetric value={plan.execConcurrency} />
                  <span className="text-primary-intense text-right font-medium">
                    {plan.priceMonthly}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        <div className="border-subtle bg-canvasSubtle border-t p-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0">
              <div className="text-basis text-sm font-medium">
                Need more than IN-XL?
              </div>
              <p className="text-muted text-xs">
                Workloads above 1K concurrency move to dedicated capacity with
                custom sizing.
              </p>
            </div>
            <a
              className="border-muted bg-canvasBase text-basis hover:bg-canvasSubtle inline-flex shrink-0 items-center gap-1.5 rounded border px-2 py-1 text-xs"
              href={pathCreator.support({ ref: 'infra-dashboard-plan' })}
            >
              <RiCalendarLine className="h-3.5 w-3.5" />
              Talk to sales
            </a>
          </div>
        </div>

        <div className="border-subtle text-muted border-t px-3 py-2 text-xs">
          Per-component sizes can also be tuned individually from each node.
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function PlanMetric({ value }: { value: string }) {
  return <span className="min-w-0 truncate">{value}</span>;
}

function InfraTierDropdown({
  onSelect,
  selectedTier,
  tiers,
}: {
  onSelect: (tierId: InfraTierId) => void;
  selectedTier: InfraTier;
  tiers: InfraTier[];
}) {
  const [open, setOpen] = useState(false);
  const triggerMetrics = getTierTriggerMetrics(selectedTier);

  const selectTier = (tierId: InfraTierId) => {
    onSelect(tierId);
    setOpen(false);
  };

  return (
    <div className="relative mx-auto mb-6 w-full max-w-xl">
      <DropdownMenu open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger asChild>
          <button
            className="border-subtle bg-canvasBase text-basis hover:bg-canvasSubtle focus:ring-primary-moderate flex w-full items-center gap-3 rounded-md border px-3 py-2 text-left shadow-sm focus:outline-none focus:ring-2"
            type="button"
          >
            <div className="grid min-w-0 flex-1 grid-cols-2 gap-3 sm:grid-cols-4">
              {triggerMetrics.map((metric) => (
                <InfoMetric
                  key={metric.label}
                  label={metric.label}
                  value={metric.value}
                />
              ))}
            </div>
            {open ? (
              <RiArrowUpSLine className="h-4 w-4 shrink-0" />
            ) : (
              <RiArrowDownSLine className="h-4 w-4 shrink-0" />
            )}
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="center"
          className="w-[min(calc(100vw-2rem),720px)] overflow-hidden p-0"
        >
          <div className="border-subtle bg-canvasSubtle text-muted border-b px-3 py-2 text-[11px] font-medium uppercase">
            Infrastructure tier
          </div>
          <div className="divide-subtle divide-y">
            {tiers.map((tier) => {
              const isSelected = tier.id === selectedTier.id;

              return (
                <button
                  className={cn(
                    'hover:bg-canvasSubtle w-full px-3 py-3 text-left',
                    isSelected && 'bg-canvasSubtle',
                  )}
                  key={tier.id}
                  onClick={() => selectTier(tier.id)}
                  type="button"
                >
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0">
                      <div className="text-basis text-sm font-medium">
                        {tier.name}
                      </div>
                      <div className="text-muted mt-0.5 text-xs">
                        {tier.description}
                      </div>
                    </div>
                    <div className="text-primary-intense shrink-0 text-xs font-medium">
                      {tier.availability}
                    </div>
                  </div>

                  <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
                    {getTierMetrics(tier).map((metric) => (
                      <TierMetric
                        key={metric.label}
                        label={metric.label}
                        value={metric.value}
                      />
                    ))}
                  </div>

                  {tier.notes?.length ? (
                    <ul className="text-muted mt-3 space-y-1 text-xs">
                      {tier.notes.map((note) => (
                        <li className="flex gap-2" key={note}>
                          <span className="text-primary-intense">-</span>
                          <span>{note}</span>
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </button>
              );
            })}
          </div>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

function getTierTriggerMetrics(tier: InfraTier) {
  const metrics = [
    { label: 'Infra tier', value: tier.name },
    ...getTierMetrics(tier),
  ];

  return metrics.slice(0, 4);
}

function getTierMetrics(tier: InfraTier) {
  return [
    { label: 'Availability', value: tier.sla },
    { label: 'P99 SLO', value: tier.dispatchP99 },
    tier.compliance ? { label: 'Compliance', value: tier.compliance } : null,
  ].filter((metric): metric is { label: string; value: string } =>
    Boolean(metric),
  );
}

function TierMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-muted truncate text-[11px] font-medium uppercase">
        {label}
      </div>
      <div className="text-basis truncate text-xs font-medium">{value}</div>
    </div>
  );
}

function InfraFlowPanel({
  backlogDepth,
  currentConcurrency,
  fetching,
  placeholders,
}: {
  backlogDepth: number;
  currentConcurrency: number;
  fetching: boolean;
  placeholders: InfraDashboardPlaceholders;
}) {
  const [selectedTierId, setSelectedTierId] = useState<InfraTierId>(
    placeholders.defaultInfraTierId,
  );
  const selectedTier =
    placeholders.infraTiers.find((tier) => tier.id === selectedTierId) ??
    placeholders.infraTiers[0];

  return (
    <section className="border-subtle bg-canvasSubtle relative mb-8 overflow-hidden rounded-md border p-4 md:p-6">
      <div
        className="absolute inset-0 opacity-80"
        style={{
          backgroundImage:
            'radial-gradient(circle, rgba(120,120,120,0.22) 1px, transparent 1px)',
          backgroundSize: '18px 18px',
        }}
      />
      <InfraTierDropdown
        tiers={placeholders.infraTiers}
        selectedTier={selectedTier}
        onSelect={setSelectedTierId}
      />

      <div className="relative grid items-center gap-4 lg:grid-cols-[1fr_56px_1fr_56px_1fr]">
        <FlowNode
          fetching={fetching}
          label="Event stream"
          primaryLabel="Rate limit | GPS"
          primaryValue={String(placeholders.eventRateLimit.current)}
          limit={placeholders.eventRateLimit.limit}
        />
        <Connector />
        <FlowNode
          accent
          fetching={fetching}
          label="Queue"
          primaryLabel="Current backlog"
          primaryValue={formatCompactNumber(backlogDepth)}
          limit={100_000}
        />
        <Connector />
        <FlowNode
          fetching={fetching}
          label="Executors"
          primaryLabel="Concurrency"
          primaryValue={formatCompactNumber(currentConcurrency)}
          limit={placeholders.functionRateLimit.limit}
        />
      </div>
    </section>
  );
}

function FlowNode({
  accent = false,
  fetching,
  label,
  limit,
  primaryLabel,
  primaryValue,
}: {
  accent?: boolean;
  fetching: boolean;
  label: string;
  limit: number;
  primaryLabel: string;
  primaryValue: string;
}) {
  const numericPrimary = Number(primaryValue.replace(/[^\d.]/g, ''));
  const progress =
    Number.isFinite(numericPrimary) && limit
      ? Math.max(8, Math.min(100, (numericPrimary / limit) * 100))
      : 28;

  return (
    <div className="bg-canvasBase border-subtle min-h-[132px] rounded-md border p-5 shadow-sm">
      <div className="text-basis mb-4 text-sm font-medium">{label}</div>
      <div className="text-muted mb-1 text-xs uppercase">{primaryLabel}</div>
      {fetching ? (
        <Skeleton className="mb-2 h-6 w-20" />
      ) : (
        <div className="text-basis mb-2 flex items-baseline justify-between">
          <span className="text-xl font-medium">{primaryValue}</span>
          <span className="text-muted text-sm">
            / {formatCompactNumber(limit)}
          </span>
        </div>
      )}
      <div className="bg-canvasMuted h-1 overflow-hidden rounded-full">
        <div
          className={cn(
            'h-full rounded-full',
            accent ? 'bg-secondary-moderate' : 'bg-primary-moderate',
          )}
          style={{ width: `${progress}%` }}
        />
      </div>
    </div>
  );
}

function Connector() {
  return (
    <div className="hidden items-center justify-center lg:flex">
      <div className="border-subtle relative h-px w-full border-t">
        <span className="border-subtle bg-canvasSubtle absolute right-0 top-1/2 h-2 w-2 -translate-y-1/2 rotate-45 border-r border-t" />
      </div>
    </div>
  );
}

function InfoMetric({
  label,
  value,
  warning,
}: {
  label: string;
  value: string;
  warning?: boolean;
}) {
  return (
    <div className="min-w-0">
      <div className="text-muted truncate text-xs uppercase">{label}</div>
      <div
        className={cn(
          'text-basis truncate text-sm font-medium',
          warning && 'text-tertiary-intense',
        )}
      >
        {value}
      </div>
    </div>
  );
}

function MostRanFunctions({
  envSlug,
  fetching,
  rows,
}: {
  envSlug: string;
  fetching: boolean;
  rows: ReturnType<typeof useInfraDashboardData>['data']['topFunctions'];
}) {
  const navigate = useNavigate();
  const functionPathCreator = useMemo(
    () => ({
      app: ({ externalAppID }: { externalAppID: string }) =>
        pathCreator.app({ envSlug, externalAppID }),
      eventType: ({ eventName }: { eventName: string }) =>
        pathCreator.eventType({ envSlug, eventName }),
      function: ({ functionSlug }: { functionSlug: string }) =>
        pathCreator.function({ envSlug, functionSlug }),
    }),
    [envSlug],
  );
  const rowsByID = useMemo(() => {
    return new Map(rows.map((row) => [row.id, row] as const));
  }, [rows]);
  const getFunctionVolume = useCallback(
    async ({ functionID }: { functionID: string }) => {
      const row = rowsByID.get(functionID);

      return {
        failureRate: row?.failureRate,
        usage: row?.usage,
      };
    },
    [rowsByID],
  );
  const columns = useFunctionColumns({
    getFunctionVolume,
    pathCreator: functionPathCreator,
  });

  return (
    <section className="mb-8">
      <div className="mb-2 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-basis text-lg font-medium">
            Most ran functions (24h)
          </h2>
        </div>
        <Link
          className="text-primary-intense inline-flex items-center gap-1 text-sm"
          to={pathCreator.functions({ envSlug })}
        >
          <RiArrowRightUpLine className="h-3.5 w-3.5" />
          View all functions
        </Link>
      </div>
      <div className="border-subtle overflow-x-auto rounded-md border">
        <div className="min-w-[860px]">
          <Table
            blankState="No function runs found for this period."
            columns={columns}
            data={rows}
            getRowHref={(row) =>
              pathCreator.function({
                envSlug,
                functionSlug: row.original.slug,
              })
            }
            isLoading={fetching}
            onRowClick={(row) => {
              navigate({
                to: pathCreator.function({
                  envSlug,
                  functionSlug: row.original.slug,
                }),
              });
            }}
          />
        </div>
      </div>
    </section>
  );
}
