'use client';

import { ContextualBanner } from '@inngest/components/Banner';
import { Button } from '@inngest/components/Button';

import { trackEvent, useTrackingUser } from '@/utils/tracking';
import { pathCreator } from '@/utils/urls';
import { useExecutionOverage } from './useExecutionOverage';

export function ExecutionOverageBanner() {
  const { isBannerVisible, executionOverageData, dismiss } = useExecutionOverage();
  const trackingUser = useTrackingUser();

  if (!isBannerVisible || !executionOverageData) {
    return null;
  }

  const handleCTAClick = () => {
    if (trackingUser) {
      trackEvent({
        name: 'app/upsell.execution.overage.cta.clicked',
        data: {
          variant: 'banner',
          executionCount: executionOverageData.executionCount,
          executionLimit: executionOverageData.executionLimit,
        },
        user: trackingUser,
        v: '2025-07-14.1',
      });
    }
  };

  return (
    <ContextualBanner
      className="flex"
      severity="warning"
      onDismiss={() => dismiss()}
      title={
        <>
          <span className="font-semibold">
            You&apos;ve used {new Intl.NumberFormat().format(executionOverageData.executionCount)}{' '}
            executions but your plan includes{' '}
            {executionOverageData.executionLimit
              ? new Intl.NumberFormat().format(executionOverageData.executionLimit)
              : 'unlimited'}
            .
          </span>{' '}
          Upgrade to avoid disruptions.
        </>
      }
      cta={
        <Button
          appearance="outlined"
          href={pathCreator.billing({ tab: 'plans', ref: 'execution-overage-banner' })}
          kind="secondary"
          label="Upgrade plan"
          onClick={handleCTAClick}
        />
      }
    >
      <div />
    </ContextualBanner>
  );
}
