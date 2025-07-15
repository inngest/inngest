'use client';

import { useCallback, useEffect } from 'react';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { RiCloseLine } from '@remixicon/react';

import { trackEvent, useTrackingUser } from '@/utils/tracking';
import { pathCreator } from '@/utils/urls';
import type { SeatOverageData } from './data';

type SeatOverageToastProps = {
  data: SeatOverageData;
  onDismiss?: () => void;
};

export function SeatOverageToast({ data, onDismiss }: SeatOverageToastProps) {
  const trackingUser = useTrackingUser();

  const handleCTAClick = useCallback(() => {
    if (trackingUser) {
      trackEvent({
        name: 'app/upsell.seat.overage.cta.clicked',
        data: {
          variant: 'toast',
          userCount: data.userCount,
          userLimit: data.userLimit,
        },
        user: trackingUser,
        v: '2025-07-14.1',
      });
    }
  }, [trackingUser, data.userCount, data.userLimit]);

  const handleDismiss = useCallback(() => {
    if (trackingUser) {
      trackEvent({
        name: 'app/upsell.seat.overage.dismissed',
        data: {
          variant: 'toast',
          userCount: data.userCount,
          userLimit: data.userLimit,
        },
        user: trackingUser,
        v: '2025-07-14.1',
      });
    }
    onDismiss?.();
  }, [trackingUser, data.userCount, data.userLimit, onDismiss]);

  // Track when toast is shown
  useEffect(() => {
    if (trackingUser) {
      trackEvent({
        name: 'app/upsell.seat.overage.prompt.shown',
        data: {
          variant: 'toast',
          userCount: data.userCount,
          userLimit: data.userLimit,
        },
        user: trackingUser,
        v: '2025-07-14.1',
      });
    }
  }, [trackingUser, data.userCount, data.userLimit]);

  return (
    <div className="bg-canvasBase border-subtle absolute bottom-0 right-0 mb-6 mr-4 w-[430px] rounded border">
      <div className="gap-x flex flex-row items-center justify-between p-3">
        <div className="text-sm leading-tight">
          <span className="font-semibold">
            You&apos;re using {data.userCount} seats but your plan includes {data.userLimit}.
          </span>
        </div>
        <Button
          icon={<RiCloseLine className="text-subtle h-5 w-5" />}
          kind="secondary"
          appearance="ghost"
          size="small"
          className="ml-.5"
          onClick={handleDismiss}
        />
      </div>
      <div className="text-muted px-3 pb-3 text-sm">Upgrade to avoid disruptions.</div>
      <div className="border-subtle border-t px-3 py-2">
        <Link
          href={pathCreator.billing({
            tab: 'plans',
            ref: 'seat-overage-toast',
          })}
          arrowOnHover={true}
          onClick={handleCTAClick}
        >
          Upgrade plan
        </Link>
      </div>
    </div>
  );
}
