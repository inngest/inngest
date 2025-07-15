'use client';

import { useEffect } from 'react';
import { ContextualBanner } from '@inngest/components/Banner';
import { Button } from '@inngest/components/Button';

import { trackEvent, useTrackingUser } from '@/utils/tracking';
import { pathCreator } from '@/utils/urls';
import type { SeatOverageData } from './data';

type SeatOverageBannerProps = {
  data: SeatOverageData;
  onDismiss?: () => void;
};

export function SeatOverageBanner({ data, onDismiss }: SeatOverageBannerProps) {
  const trackingUser = useTrackingUser();

  const handleCTAClick = () => {
    if (trackingUser) {
      trackEvent({
        name: 'app/upsell.seat.overage.cta.clicked',
        data: {
          variant: 'banner',
          userCount: data.userCount,
          userLimit: data.userLimit,
        },
        user: trackingUser,
        v: '2025-07-14.1',
      });
    }
  };

  // Track when banner is shown (only once when component mounts)
  useEffect(() => {
    if (trackingUser) {
      trackEvent({
        name: 'app/upsell.seat.overage.prompt.shown',
        data: {
          variant: 'banner',
          userCount: data.userCount,
          userLimit: data.userLimit,
        },
        user: trackingUser,
        v: '2025-07-14.1',
      });
    }
  }, [trackingUser, data.userCount, data.userLimit]);

  return (
    <ContextualBanner
      className="flex"
      severity="warning"
      onDismiss={onDismiss}
      title={
        <>
          <span className="font-semibold">
            You&apos;re using {data.userCount} seats but your plan includes {data.userLimit}.
          </span>{' '}
          Upgrade now and keep everyone on the team.
        </>
      }
      cta={
        <Button
          appearance="outlined"
          href={pathCreator.billing({ tab: 'plans', ref: 'seat-overage-banner' })}
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
