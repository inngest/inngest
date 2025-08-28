'use client';

import { useEffect } from 'react';
import { ContextualBanner } from '@inngest/components/Banner';
import { Button } from '@inngest/components/Button';

import { trackEvent, useTrackingUser } from '@/utils/tracking';
import { pathCreator } from '@/utils/urls';
import { useExecutionOverage } from './useExecutionOverage';

export function ExecutionOverageBanner() {
  const { isBannerVisible, executionOverageData, dismiss } = useExecutionOverage();
  const trackingUser = useTrackingUser();

  // Track CTA viewed when banner becomes visible (temporarily disabled).
  // useEffect(() => {
  //   if (isBannerVisible && executionOverageData && trackingUser) {
  //     trackEvent({
  //       name: 'app/billing.cta.viewed',
  //       data: {
  //         cta: 'execution-overage-banner',
  //         entitlement: 'executions',
  //       },
  //       user: trackingUser,
  //       v: '2025-01-15.1',
  //     });
  //   }
  // }, [isBannerVisible, executionOverageData, trackingUser]);

  if (!isBannerVisible || !executionOverageData) {
    return null;
  }

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
        />
      }
    >
      <div />
    </ContextualBanner>
  );
}
