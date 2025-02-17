import type { Severity } from '@inngest/components/Banner';

import type { EntitlementUsageQuery } from '@/gql/graphql';
import { BillingBannerTooltip } from './BillingBannerTooltip';

export function parseEntitlementUsage(data: EntitlementUsageQuery['account']['entitlements']): {
  bannerMessage: React.ReactNode;
  bannerSeverity: Severity;
  items: [string, React.ReactNode][];
} {
  const { runCount, concurrency, stepCount } = data;
  const issues = new Issues();

  // Users who can buy additional runs should not warnings about nearing the run
  // limit.
  if (runCount.limit && !runCount.overageAllowed) {
    if (runCount.usage >= runCount.limit) {
      issues.add(
        'run_count',
        <div className="flex items-center">
          {Intl.NumberFormat().format(runCount.usage)} /{' '}
          {Intl.NumberFormat().format(runCount.limit)} function runs
          <BillingBannerTooltip>
            Exceeding the function run limit may result in service disruption.
          </BillingBannerTooltip>
        </div>,
        IssueSeverity.hardLimitReached
      );
    } else if (runCount.usage >= runCount.limit * 0.8) {
      issues.add(
        'run_count',
        <div className="flex items-center">
          {Intl.NumberFormat().format(runCount.usage)} /{' '}
          {Intl.NumberFormat().format(runCount.limit)} function runs
          <BillingBannerTooltip>
            Exceeding the function run limit may result in service disruption.
          </BillingBannerTooltip>
        </div>,
        IssueSeverity.hardLimitNear
      );
    }
  }

  // Users who can buy additional steps should not warnings about nearing the
  // step limit.
  if (stepCount.limit && !stepCount.overageAllowed) {
    if (stepCount.usage >= stepCount.limit) {
      issues.add(
        'step_count',
        <div className="flex items-center">
          {Intl.NumberFormat().format(stepCount.usage)} /{' '}
          {Intl.NumberFormat().format(stepCount.limit)} steps
          <BillingBannerTooltip>
            Exceeding the step limit may result in service disruption.
          </BillingBannerTooltip>
        </div>,
        IssueSeverity.hardLimitReached
      );
    } else if (stepCount.usage >= stepCount.limit * 0.8) {
      issues.add(
        'step_count',
        <div className="flex items-center">
          {Intl.NumberFormat().format(stepCount.usage)} /{' '}
          {Intl.NumberFormat().format(stepCount.limit)} steps
          <BillingBannerTooltip>
            Exceeding the step limit may result in service disruption.
          </BillingBannerTooltip>
        </div>,
        IssueSeverity.hardLimitNear
      );
    }
  }

  if (concurrency.usage >= 12) {
    issues.add(
      'concurrency',
      <div className="flex items-center">
        Account concurrency limit reached in {concurrency.usage} of the past 24 hours
        <BillingBannerTooltip>
          Reaching the concurrency limit adds delays between steps, making function runs take longer
          to complete.
        </BillingBannerTooltip>
      </div>,
      IssueSeverity.softLimitReached
    );
  }

  return {
    bannerMessage: issues.getBannerMessage(),
    bannerSeverity: issues.getBannerSeverity(),
    items: issues.getItems(),
  };
}

const IssueSeverity = {
  softLimitReached: 0,
  hardLimitNear: 1,
  hardLimitReached: 2,
} as const;
type IssueSeverity = (typeof IssueSeverity)[keyof typeof IssueSeverity];

class Issues {
  private items: Record<string, React.ReactNode> = {};
  private maxIssueSeverity: IssueSeverity = IssueSeverity.softLimitReached;

  add(key: string, message: React.ReactNode, severity: IssueSeverity) {
    this.items[key] = message;

    if (severity > this.maxIssueSeverity) {
      this.maxIssueSeverity = severity;
    }
  }

  getBannerMessage(): React.ReactNode {
    if (this.maxIssueSeverity === IssueSeverity.hardLimitNear) {
      return (
        <>
          <span className="font-semibold">High usage.</span> You are nearing the usage included in
          your plan. Please upgrade to avoid service disruption.
        </>
      );
    } else if (this.maxIssueSeverity === IssueSeverity.hardLimitReached) {
      return (
        <>
          <span className="font-semibold">Limit exceeded.</span> You have exceeded the usage
          included in your plan. Please upgrade to avoid service disruption.
        </>
      );
    } else {
      return (
        <>
          <span className="font-semibold">Limit reached.</span> Performance may be impacted.
        </>
      );
    }
  }

  getBannerSeverity(): Severity {
    if (this.maxIssueSeverity === IssueSeverity.hardLimitReached) {
      return 'error';
    }

    return 'warning';
  }

  getItems(): [string, React.ReactNode][] {
    return Object.entries(this.items);
  }
}
