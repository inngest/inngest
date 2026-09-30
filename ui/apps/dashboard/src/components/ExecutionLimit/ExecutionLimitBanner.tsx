import { Banner } from '@inngest/components/Banner';

import { upgradeLinkProps, useExecutionLimit } from './useExecutionLimit';

const numberFormatter = new Intl.NumberFormat('en-US');

export function ExecutionLimitBanner() {
  const data = useExecutionLimit();

  if (!data?.enhanced || data.band !== 'limitReached') return null;

  const formattedUsage = numberFormatter.format(data.usage);

  return (
    <Banner
      severity="error"
      cta={
        <Banner.Link
          severity="error"
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
      New runs are paused. You&apos;ve used {formattedUsage} executions this
      month and reached the Hobby limit. Upgrade to Pro to resume now.
    </Banner>
  );
}
