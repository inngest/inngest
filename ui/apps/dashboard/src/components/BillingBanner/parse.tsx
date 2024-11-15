import type { EntitlementUsageQuery } from '@/gql/graphql';
import type { Severity } from '../Banner';

export function parseEntitlementUsage(data: EntitlementUsageQuery['account']['entitlementUsage']): {
  bannerMessage: React.ReactNode;
  bannerSeverity: Severity;
  items: [string, React.ReactNode][];
} {
  const { runCount, accountConcurrencyLimitHits } = data;
  const issues = new Issues();

  if (runCount.limit) {
    if (runCount.current >= runCount.limit) {
      issues.add(
        'run_count',
        <>
          <span className="font-semibold">{Intl.NumberFormat().format(runCount.current)}</span> /{' '}
          {Intl.NumberFormat().format(runCount.limit)} function runs
        </>,
        IssueSeverity.limitExceeded
      );
    } else if (runCount.current >= runCount.limit * 0.8) {
      issues.add(
        'run_count',
        <>
          <span className="font-semibold">{Intl.NumberFormat().format(runCount.current)}</span> /{' '}
          {Intl.NumberFormat().format(runCount.limit)} function runs
        </>,
        IssueSeverity.nearingLimit
      );
    }
  }
  if (accountConcurrencyLimitHits >= 12) {
    issues.add(
      'concurrency',
      <>
        <span className="font-semibold">
          Account concurrency limit reached in {accountConcurrencyLimitHits}
        </span>{' '}
        of the past 24 hours
      </>,
      IssueSeverity.nearingLimit
    );
  }

  return {
    bannerMessage: issues.getBannerMessage(),
    bannerSeverity: issues.getBannerSeverity(),
    items: issues.getItems(),
  };
}

const IssueSeverity = {
  nearingLimit: 0,
  limitExceeded: 1,
} as const;
type IssueSeverity = (typeof IssueSeverity)[keyof typeof IssueSeverity];

class Issues {
  private items: Record<string, React.ReactNode> = {};
  private maxIssueSeverity: IssueSeverity = IssueSeverity.nearingLimit;

  add(key: string, message: React.ReactNode, severity: IssueSeverity) {
    this.items[key] = message;

    if (severity > this.maxIssueSeverity) {
      this.maxIssueSeverity = severity;
    }
  }

  getBannerMessage(): React.ReactNode {
    if (this.maxIssueSeverity === IssueSeverity.nearingLimit) {
      return (
        <>
          <span className="font-semibold">High usage.</span> You are nearing the usage included in
          your plan. Please upgrade to avoid service disruption.
        </>
      );
    }

    return (
      <>
        <span className="font-semibold">Limit exceeded.</span> You have exceeded the usage included
        in your plan. Please upgrade to avoid service disruption.
      </>
    );
  }

  getBannerSeverity(): Severity {
    if (this.maxIssueSeverity === IssueSeverity.nearingLimit) {
      return 'warning';
    }

    return 'error';
  }

  getItems(): [string, React.ReactNode][] {
    return Object.entries(this.items);
  }
}
