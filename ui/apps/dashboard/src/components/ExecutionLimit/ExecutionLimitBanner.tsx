import { Banner, type Severity } from '@inngest/components/Banner';

import type { UsageBand } from './executionLimit';
import { useDismissal } from './useDismissal';
import { upgradeLinkProps, useExecutionLimit } from './useExecutionLimit';

const bannerSeverity: Partial<Record<UsageBand, Severity>> = {
  '50': 'info',
  '75': 'warning',
};

export function ExecutionLimitBanner() {
  const data = useExecutionLimit();
  const { isReady, isDismissed, dismiss } = useDismissal(
    'banner',
    data?.accountID,
  );

  if (!data?.enhanced || !isReady || isDismissed) return null;

  const severity = bannerSeverity[data.band];
  if (!severity) return null;

  return (
    <Banner
      severity={severity}
      onDismiss={dismiss}
      cta={
        <Banner.Link
          severity={severity}
          className="mr-2 shrink-0"
          {...upgradeLinkProps(
            data.marketplaceBillingURL,
            'app-hobby-execution-limit-banner',
          )}
        >
          Upgrade
        </Banner.Link>
      }
    >
      You have used over 50% of your plan&apos;s executions. Upgrade your plan
      to avoid disruptions
    </Banner>
  );
}
