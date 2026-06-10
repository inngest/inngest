import { useCallback, useMemo, useState } from 'react';
import { useOrganization } from '@clerk/tanstack-react-start';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { useColumns as useFunctionColumns } from '@inngest/components/Functions/columns';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Table } from '@inngest/components/Table';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiArrowDownSLine,
  RiArrowRightUpLine,
  RiArrowUpSLine,
  RiCalendarLine,
  RiCheckboxCircleLine,
  RiGroupLine,
  RiShieldCheckLine,
} from '@remixicon/react';
import { Link, useNavigate, useRouter } from '@tanstack/react-router';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import UpdateCardModal from '@/components/Billing/BillingDetails/UpdateCardModal';
import CheckoutModal, {
  type CheckoutItem,
} from '@/components/Billing/Plans/CheckoutModal';
import ConfirmPlanChangeModal from '@/components/Billing/Plans/ConfirmPlanChangeModal';
import { useEnvironment } from '@/components/Environments/environment-context';
import { UpdateAccountAddonQuantityDocument } from '@/gql/graphql';
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
  formatCentsMonthly,
  formatPercent,
  getCurrentInfraTierId,
  getInfraPlanBillingAction,
  type BillingPlanSource,
  type InfraConcurrencyAddonSource,
  type InfraPlanAddonUpdate,
} from './utils';

function formatCacheAge(cachedAt: number) {
  const ageMs = Math.max(0, Date.now() - cachedAt);
  const minutes = Math.floor(ageMs / 60_000);

  if (minutes < 5) {
    return null;
  }

  return `${minutes}m`;
}

export function InfraDashboard() {
  const env = useEnvironment();
  const { cacheStatus, data, loading, refetchBillingData } =
    useInfraDashboardData(TIME_RANGE_OPTIONS[0]);
  const cacheAge = cacheStatus.cachedAt
    ? formatCacheAge(cacheStatus.cachedAt)
    : null;
  const placeholders = data.placeholders;
  const billingDays = billingCycleDaysRemaining();
  const selectedPlan = data.currentInfraPlan;

  return (
    <div className="bg-canvasBase min-h-full w-full px-4 pb-16 pt-4 lg:px-6">
      <header className="mb-4 flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-basis text-xl font-medium leading-7">
              {env.name}
            </h1>
          </div>
          {cacheStatus.isUsingCachedData && cacheAge ? (
            <div className="text-muted/50 mt-1 text-xs">
              Cached {cacheAge} ago · refreshing
            </div>
          ) : null}
        </div>

        <div className="flex w-fit max-w-full flex-col items-stretch gap-2">
          <InfraPlanDropdown
            billingActionsReady={data.billingActionsReady}
            billingPlanReady={data.billingPlanReady}
            concurrencyAddon={data.concurrencyAddon}
            currentBillingPlan={data.currentBillingPlan}
            currentConcurrencyLimit={data.accountConcurrencyLimit}
            currentPlanSku={data.currentInfraPlanSku}
            hasPaymentMethod={data.hasPaymentMethod}
            onBillingChange={refetchBillingData}
            isEnterprisePlan={data.isEnterprisePlan}
            plans={data.infraPlans}
            proPlanAmountCents={data.proPlanAmountCents}
            selectedPlan={selectedPlan}
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

      <section className="mb-4 grid grid-cols-1 gap-2 md:grid-cols-3">
        <KpiCard
          fetching={loading.eventsReceived}
          label="Events received"
          value={formatCompactNumber(data.eventsReceived)}
        />
        <KpiCard
          fetching={loading.executionsRan}
          label="Executions ran (runs + steps)"
          value={formatCompactNumber(data.executionsRan)}
        />
        <KpiCard
          fetching={loading.backlog}
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
        currentInfraTierId={
          data.isEnterprisePlan
            ? 'dedicated'
            : data.billingPlanReady && data.currentInfraPlan.isCurrent
            ? getCurrentInfraTierId(data.currentInfraPlanSku)
            : undefined
        }
        eventsReceived={data.eventsReceived}
        eventsFetching={loading.eventsReceived}
        infraPlan={data.currentInfraPlan}
        isEnterprisePlan={data.isEnterprisePlan}
        placeholders={placeholders}
        queueFetching={loading.queue}
        executorsFetching={loading.executors}
      />

      <MostRanFunctions
        envSlug={env.slug}
        rows={data.topFunctions}
        fetching={loading.topFunctions}
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
  billingActionsReady,
  billingPlanReady,
  concurrencyAddon,
  currentBillingPlan,
  currentConcurrencyLimit,
  currentPlanSku,
  hasPaymentMethod,
  isEnterprisePlan,
  onBillingChange,
  plans,
  proPlanAmountCents,
  selectedPlan,
}: {
  billingActionsReady: boolean;
  billingPlanReady: boolean;
  concurrencyAddon?: InfraConcurrencyAddonSource | null;
  currentBillingPlan?: BillingPlanSource | null;
  currentConcurrencyLimit?: number | null;
  currentPlanSku: InfraPlanSku;
  hasPaymentMethod: boolean;
  isEnterprisePlan: boolean;
  onBillingChange: () => Promise<void>;
  plans: InfraPlan[];
  proPlanAmountCents?: number | null;
  selectedPlan: InfraPlan;
}) {
  const router = useRouter();
  const { isLoaded: orgLoaded, membership } = useOrganization();
  const canChangePlan = orgLoaded && membership?.role === 'org:admin';
  const [open, setOpen] = useState(false);
  const [checkoutData, setCheckoutData] = useState<{
    followUpAddonUpdate: InfraPlanAddonUpdate | null;
    items: CheckoutItem[];
  } | null>(null);
  const [planChangeData, setPlanChangeData] = useState<{
    action: 'cancel';
    addonUpdate: InfraPlanAddonUpdate | null;
    items: CheckoutItem[];
  } | null>(null);
  const [pendingAddonUpdate, setPendingAddonUpdate] =
    useState<InfraPlanAddonUpdate | null>(null);
  const [showPaymentMethodModal, setShowPaymentMethodModal] = useState(false);
  const [isUpdatingAddon, setIsUpdatingAddon] = useState(false);

  const [, updateAccountAddonQuantity] = useMutation(
    UpdateAccountAddonQuantityDocument,
  );

  const handleBillingSuccess = useCallback(async () => {
    await onBillingChange();
    await router.invalidate();
    toast.success('Plan changed successfully');
  }, [onBillingChange, router]);

  const handleCheckoutSuccess = useCallback(() => {
    const followUpAddonUpdate = checkoutData?.followUpAddonUpdate ?? null;
    setCheckoutData(null);

    if (followUpAddonUpdate) {
      void onBillingChange().finally(() => {
        setPendingAddonUpdate(followUpAddonUpdate);
      });
      return;
    }

    void handleBillingSuccess();
  }, [
    checkoutData?.followUpAddonUpdate,
    handleBillingSuccess,
    onBillingChange,
  ]);

  const handleAddonUpdate = useCallback(async () => {
    if (!pendingAddonUpdate) {
      return;
    }

    setIsUpdatingAddon(true);
    const result = await updateAccountAddonQuantity({
      addonName: pendingAddonUpdate.addonName,
      quantity: pendingAddonUpdate.addonQuantity,
    });
    setIsUpdatingAddon(false);

    if (result.error) {
      toast.error('Failed to update concurrency add-on');
      return;
    }

    setPendingAddonUpdate(null);
    await handleBillingSuccess();
  }, [handleBillingSuccess, pendingAddonUpdate, updateAccountAddonQuantity]);

  const handlePlanClick = useCallback(
    (plan: InfraPlan) => {
      if (!canChangePlan || !billingActionsReady) {
        return;
      }

      const action = getInfraPlanBillingAction({
        concurrencyAddon,
        currentConcurrencyLimit,
        currentPlan: currentBillingPlan,
        currentPlanSku,
        proPlanAmountCents,
        targetSku: plan.sku,
      });

      if (action.type === 'current') {
        return;
      }

      if (action.type === 'unavailable') {
        toast.error(action.reason);
        return;
      }

      setOpen(false);

      if (action.type === 'cancel-to-free') {
        setPlanChangeData({
          action: 'cancel',
          addonUpdate: action.addonUpdate,
          items: [action.item],
        });
        return;
      }

      if (action.type === 'upgrade-base-plan') {
        setCheckoutData({
          followUpAddonUpdate: action.addonUpdate,
          items: [action.item],
        });
        return;
      }

      if (action.isIncrease && !hasPaymentMethod) {
        setPendingAddonUpdate(action);
        setShowPaymentMethodModal(true);
        return;
      }

      setPendingAddonUpdate(action);
    },
    [
      canChangePlan,
      billingActionsReady,
      concurrencyAddon,
      currentBillingPlan,
      currentConcurrencyLimit,
      currentPlanSku,
      hasPaymentMethod,
      proPlanAmountCents,
    ],
  );

  return (
    <>
      <DropdownMenu
        open={isEnterprisePlan ? false : open}
        onOpenChange={(nextOpen) => {
          if (!isEnterprisePlan) {
            setOpen(nextOpen);
          }
        }}
      >
        <DropdownMenuTrigger asChild>
          <button
            className={cn(
              'border-muted bg-canvasBase text-basis flex max-w-full items-center justify-between gap-2 rounded border px-2 py-1 text-xs disabled:cursor-default disabled:opacity-100',
              !isEnterprisePlan &&
                'hover:bg-canvasSubtle focus:ring-primary-moderate focus:outline-none focus:ring-2',
            )}
            disabled={isEnterprisePlan}
            type="button"
          >
            <span className="flex min-w-0 flex-wrap items-center gap-2">
              <span className="bg-canvasMuted rounded px-1.5 py-0.5 font-medium">
                {selectedPlan.displaySku ?? selectedPlan.sku}
              </span>
              <span>{selectedPlan.eventStream}</span>
              <span className="text-disabled">·</span>
              <span>{selectedPlan.queueDepth} depth</span>
              <span className="text-disabled">·</span>
              <span>{selectedPlan.execConcurrency} concurrency</span>
            </span>
            {!isEnterprisePlan ? (
              open ? (
                <RiArrowUpSLine className="h-3.5 w-3.5 shrink-0" />
              ) : (
                <RiArrowDownSLine className="h-3.5 w-3.5 shrink-0" />
              )
            ) : null}
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
                <span>Events</span>
                <span>Queue depth</span>
                <span>Exec concurrency</span>
                <span className="text-right">Price / mo</span>
              </div>
              {plans.map((plan) => {
                const action = billingActionsReady
                  ? getInfraPlanBillingAction({
                      concurrencyAddon,
                      currentConcurrencyLimit,
                      currentPlan: currentBillingPlan,
                      currentPlanSku,
                      proPlanAmountCents,
                      targetSku: plan.sku,
                    })
                  : ({
                      reason: 'Billing plan is still loading.',
                      type: 'unavailable',
                    } as const);
                const isCurrent = billingPlanReady && action.type === 'current';
                const isActionable =
                  billingActionsReady &&
                  canChangePlan &&
                  action.type !== 'current' &&
                  action.type !== 'unavailable';

                return (
                  <button
                    className={cn(
                      'border-subtle text-basis grid w-full grid-cols-[96px_140px_140px_160px_1fr] items-center border-t px-3 py-2.5 text-left text-xs disabled:cursor-default disabled:opacity-100',
                      isCurrent && 'bg-canvasSubtle',
                      isActionable &&
                        'hover:bg-canvasSubtle focus:bg-canvasSubtle focus:outline-none',
                    )}
                    disabled={!isActionable}
                    key={plan.sku}
                    onClick={() => handlePlanClick(plan)}
                    title={
                      !canChangePlan
                        ? 'Only organization admins can change plans.'
                        : action.type === 'unavailable'
                        ? action.reason
                        : undefined
                    }
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
                    <span className="flex min-w-0 items-center justify-end gap-2">
                      {isCurrent ? <YourPlanBadge /> : null}
                      {!isCurrent ? (
                        <span className="text-primary-intense truncate text-right font-medium">
                          {plan.priceMonthly}
                        </span>
                      ) : null}
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
        </DropdownMenuContent>
      </DropdownMenu>

      {checkoutData ? (
        <CheckoutModal
          items={checkoutData.items}
          onCancel={() => setCheckoutData(null)}
          onSuccess={handleCheckoutSuccess}
        />
      ) : null}
      {planChangeData ? (
        <ConfirmPlanChangeModal
          action={planChangeData.action}
          items={planChangeData.items}
          onCancel={() => setPlanChangeData(null)}
          onBeforePlanChange={async () => {
            if (!planChangeData.addonUpdate) {
              return;
            }

            const result = await updateAccountAddonQuantity({
              addonName: planChangeData.addonUpdate.addonName,
              quantity: planChangeData.addonUpdate.addonQuantity,
            });

            if (result.error) {
              throw result.error;
            }
          }}
          onSuccess={() => {
            setPlanChangeData(null);
            void handleBillingSuccess();
          }}
        />
      ) : null}
      {showPaymentMethodModal ? (
        <UpdateCardModal
          onCancel={() => {
            setShowPaymentMethodModal(false);
            setPendingAddonUpdate(null);
          }}
          onSuccess={() => {
            setShowPaymentMethodModal(false);
            void onBillingChange();
          }}
        />
      ) : null}
      {pendingAddonUpdate && !showPaymentMethodModal ? (
        <AlertModal
          autoClose={false}
          className="w-full max-w-md"
          confirmButtonKind="primary"
          confirmButtonLabel={
            pendingAddonUpdate.estimatedMonthlyAddonCost > 0
              ? 'Confirm and pay'
              : 'Confirm'
          }
          description={getAddonUpdateDescription(pendingAddonUpdate)}
          isLoading={isUpdatingAddon}
          isOpen={true}
          onClose={() => setPendingAddonUpdate(null)}
          onSubmit={handleAddonUpdate}
          title={getAddonUpdateTitle(pendingAddonUpdate)}
        />
      ) : null}
    </>
  );
}

function PlanMetric({ value }: { value: string }) {
  return <span className="min-w-0 truncate">{value}</span>;
}

function getAddonUpdateTitle(update: InfraPlanAddonUpdate) {
  return `${update.isIncrease ? 'Upgrade' : 'Downgrade'} to ${
    update.targetSku
  }`;
}

function getAddonUpdateDescription(update: InfraPlanAddonUpdate) {
  const totalCost = formatCentsMonthly(update.targetMonthlyAmountCents);

  return `Your plan will change to ${update.targetSku} for ${
    totalCost || '$0'
  }/mo.`;
}

function YourPlanBadge({ label = 'Your plan' }: { label?: string }) {
  return (
    <span className="bg-primary-intense text-alwaysWhite inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium">
      <RiCheckboxCircleLine className="h-3.5 w-3.5" />
      {label}
    </span>
  );
}

function InfraTierDropdown({
  currentTierId,
  disabled = false,
  isEnterprisePlan = false,
  selectedTier,
  tiers,
}: {
  currentTierId?: InfraTierId;
  disabled?: boolean;
  isEnterprisePlan?: boolean;
  selectedTier: InfraTier;
  tiers: InfraTier[];
}) {
  const [open, setOpen] = useState(false);
  const triggerMetrics = getTierTriggerMetrics(selectedTier);

  return (
    <div className="relative z-10 mx-auto mb-6 w-full max-w-xl">
      <DropdownMenu
        open={disabled ? false : open}
        onOpenChange={(nextOpen) => {
          if (!disabled) {
            setOpen(nextOpen);
          }
        }}
      >
        <DropdownMenuTrigger asChild>
          <button
            className={cn(
              'border-subtle bg-canvasBase text-basis flex w-full items-center gap-3 rounded-md border px-3 py-2 text-left shadow-sm disabled:cursor-default disabled:opacity-100',
              !disabled &&
                'hover:bg-canvasSubtle focus:ring-primary-moderate focus:outline-none focus:ring-2',
            )}
            disabled={disabled}
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
            {!disabled ? (
              open ? (
                <RiArrowUpSLine className="h-4 w-4 shrink-0" />
              ) : (
                <RiArrowDownSLine className="h-4 w-4 shrink-0" />
              )
            ) : null}
          </button>
        </DropdownMenuTrigger>
        {!disabled ? (
          <DropdownMenuContent
            align="center"
            className="w-[min(calc(100vw-2rem),720px)] overflow-y-auto p-0"
            style={{
              maxHeight:
                'min(var(--radix-dropdown-menu-content-available-height), calc(100vh - 2rem))',
            }}
          >
            <div className="border-subtle bg-canvasSubtle text-muted border-b px-3 py-2 text-[11px] font-medium uppercase">
              Infrastructure tier
            </div>
            <div className="divide-subtle divide-y">
              {tiers.map((tier) => {
                const isSelected = tier.id === selectedTier.id;
                const isCurrentTier = tier.id === currentTierId;

                return (
                  <div
                    className={cn(
                      'w-full cursor-default px-3 py-3 text-left',
                      isSelected && 'bg-canvasSubtle',
                    )}
                    key={tier.id}
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
                      {isCurrentTier ? (
                        <YourPlanBadge
                          label={
                            isEnterprisePlan ? 'Current plan' : 'Your plan'
                          }
                        />
                      ) : (
                        <div className="text-primary-intense shrink-0 text-xs font-medium">
                          {tier.availability}
                        </div>
                      )}
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
                  </div>
                );
              })}
            </div>
          </DropdownMenuContent>
        ) : null}
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
  currentInfraTierId,
  eventsReceived,
  eventsFetching,
  executorsFetching,
  infraPlan,
  isEnterprisePlan,
  placeholders,
  queueFetching,
}: {
  backlogDepth: number;
  currentConcurrency: number;
  currentInfraTierId?: InfraTierId;
  eventsReceived: number;
  eventsFetching: boolean;
  executorsFetching: boolean;
  infraPlan: InfraPlan;
  isEnterprisePlan: boolean;
  placeholders: InfraDashboardPlaceholders;
  queueFetching: boolean;
}) {
  const selectedTier =
    placeholders.infraTiers.find(
      (tier) =>
        tier.id === (currentInfraTierId ?? placeholders.defaultInfraTierId),
    ) ?? placeholders.infraTiers[0];

  return (
    <section className="border-subtle bg-canvasSubtle relative z-0 mb-10 min-h-[280px] shrink-0 overflow-visible rounded-md border p-4 md:p-6">
      <div
        className="pointer-events-none absolute inset-0 rounded-md opacity-80"
        style={{
          backgroundImage:
            'radial-gradient(circle, rgba(120,120,120,0.22) 1px, transparent 1px)',
          backgroundSize: '18px 18px',
        }}
      />
      <InfraTierDropdown
        currentTierId={currentInfraTierId}
        disabled={isEnterprisePlan}
        isEnterprisePlan={isEnterprisePlan}
        tiers={placeholders.infraTiers}
        selectedTier={selectedTier}
      />

      <div className="relative z-10 grid items-center gap-4 lg:grid-cols-[1fr_56px_1fr_56px_1fr]">
        <FlowNode
          fetching={eventsFetching}
          label="Events"
          primaryLabel={
            infraPlan.eventStreamUnit === 'events'
              ? 'Events received'
              : 'Rate limit | GPS'
          }
          primaryHint={
            infraPlan.eventStreamUnit === 'events' ? 'Soft limit' : undefined
          }
          primaryValue={
            infraPlan.eventStreamUnit === 'events'
              ? formatCompactNumber(eventsReceived)
              : String(placeholders.eventRateLimit.current)
          }
          progressValue={
            infraPlan.eventStreamUnit === 'events'
              ? eventsReceived
              : placeholders.eventRateLimit.current
          }
          limit={infraPlan.eventStreamLimit}
        />
        <Connector />
        <FlowNode
          accent
          fetching={queueFetching}
          label="Queue"
          primaryLabel="Current backlog"
          primaryHint="Soft limit"
          primaryValue={formatCompactNumber(backlogDepth)}
          progressValue={backlogDepth}
          limit={infraPlan.queueDepthLimit}
        />
        <Connector />
        <FlowNode
          fetching={executorsFetching}
          label="Executors"
          primaryLabel="Concurrency in use"
          primaryHint="~ Approx."
          primaryValue={formatCompactNumber(currentConcurrency)}
          progressValue={currentConcurrency}
          limit={infraPlan.execConcurrencyLimit}
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
  progressValue,
  primaryLabel,
  primaryHint,
  primaryValue,
}: {
  accent?: boolean;
  fetching: boolean;
  label: string;
  limit: number | null;
  progressValue?: number;
  primaryLabel: string;
  primaryHint?: string;
  primaryValue: string;
}) {
  const numericPrimary =
    progressValue ?? Number(primaryValue.replace(/[^\d.]/g, ''));
  const progressPercent =
    Number.isFinite(numericPrimary) && typeof limit === 'number' && limit > 0
      ? (numericPrimary / limit) * 100
      : null;
  const progress =
    progressPercent === null ? 28 : Math.max(8, Math.min(100, progressPercent));
  const progressColor =
    progressPercent !== null && progressPercent >= 90
      ? 'bg-errorContrast'
      : progressPercent !== null && progressPercent >= 75
      ? 'bg-warning'
      : accent
      ? 'bg-secondary-moderate'
      : 'bg-primary-moderate';
  const limitLabel =
    typeof limit === 'number' ? formatCompactNumber(limit) : 'Unlimited';

  return (
    <div className="bg-canvasBase border-subtle min-h-[132px] rounded-md border p-5 shadow-sm">
      <div className="text-basis mb-4 text-sm font-medium">{label}</div>
      <div className="mb-1 flex items-center justify-between gap-3 text-xs">
        <span className="text-muted uppercase">{primaryLabel}</span>
        {primaryHint ? (
          <span className="text-muted/40 shrink-0 text-right">
            {primaryHint}
          </span>
        ) : null}
      </div>
      {fetching ? (
        <Skeleton className="mb-2 h-6 w-20" />
      ) : (
        <div className="text-basis mb-2 flex items-baseline justify-between">
          <span className="text-xl font-medium">{primaryValue}</span>
          <span className="text-muted text-sm">/ {limitLabel}</span>
        </div>
      )}
      <div className="bg-canvasMuted h-1 overflow-hidden rounded-full">
        <div
          className={cn('h-full rounded-full', progressColor)}
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
    <section className="pb-20">
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
