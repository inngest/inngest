'use client';

import { useEffect } from 'react';
import NextLink from 'next/link';
import Link from 'next/link';
import { Button } from '@inngest/components/Button';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiCloseLine, RiErrorWarningFill } from '@remixicon/react';

import { trackEvent, useTrackingUser } from '@/utils/tracking';
import { pathCreator } from '@/utils/urls';
import { useSeatOverage } from './useSeatOverage';

// TODO: turn into a component for all other upsell widgets
export default function SeatOverageWidget({ collapsed }: { collapsed: boolean }) {
  const { isWidgetVisible, seatOverageData, dismiss } = useSeatOverage();
  const trackingUser = useTrackingUser();

  const handleCTAClick = () => {
    if (trackingUser && seatOverageData) {
      trackEvent({
        name: 'app/upsell.seat.overage.cta.clicked',
        data: {
          variant: 'widget',
          userCount: seatOverageData.userCount,
          userLimit: seatOverageData.userLimit,
        },
        user: trackingUser,
        v: '2025-07-14.1',
      });
    }
  };

  const handleDismiss = () => {
    if (trackingUser && seatOverageData) {
      trackEvent({
        name: 'app/upsell.seat.overage.dismissed',
        data: {
          variant: 'widget',
          userCount: seatOverageData.userCount,
          userLimit: seatOverageData.userLimit,
        },
        user: trackingUser,
        v: '2025-07-14.1',
      });
    }
    dismiss();
  };

  // Track when widget is shown
  useEffect(() => {
    if (trackingUser && seatOverageData && isWidgetVisible) {
      trackEvent({
        name: 'app/upsell.seat.overage.prompt.shown',
        data: {
          variant: 'widget',
          userCount: seatOverageData.userCount,
          userLimit: seatOverageData.userLimit,
        },
        user: trackingUser,
        v: '2025-07-14.1',
      });
    }
  }, [trackingUser, seatOverageData, isWidgetVisible]);

  if (!isWidgetVisible || !seatOverageData) {
    return null;
  }

  return (
    <>
      {collapsed && (
        <MenuItem
          href={pathCreator.billing({
            tab: 'plans',
            ref: 'seat-overage-widget',
          })}
          className="border border-amber-200 bg-amber-50"
          collapsed={collapsed}
          text="Upgrade plan"
          icon={<RiErrorWarningFill className="h-[18px] w-[18px] text-amber-600" />}
        />
      )}

      {!collapsed && (
        <NextLink
          href={pathCreator.billing({
            ref: 'seat-overage-widget',
          })}
          className="text-basis mb-5 block rounded border border-amber-200 bg-amber-50 p-3 leading-tight"
          onClick={handleCTAClick}
        >
          <div className="flex min-h-[110px] flex-col justify-between">
            <div>
              <div className="flex items-center justify-between">
                <RiErrorWarningFill className="h-5 w-5 text-amber-600" />
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      icon={<RiCloseLine className="text-subtle" />}
                      kind="secondary"
                      appearance="ghost"
                      size="small"
                      className="hover:bg-amber-100"
                      onClick={(e) => {
                        e.preventDefault();
                        handleDismiss();
                      }}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="right" className="max-w-40">
                    <p>Dismiss for 24 hours</p>
                  </TooltipContent>
                </Tooltip>
              </div>
              <p className="flex items-center gap-1.5 text-amber-800">Seat limit exceeded</p>
              <p className="text-sm text-amber-700">
                You&apos;re using {seatOverageData.userCount} seats but your plan includes{' '}
                {seatOverageData.userLimit}.
              </p>
            </div>
            <Link
              href={pathCreator.billing({
                ref: 'seat-overage-widget',
              })}
              className="text-sm text-amber-800 hover:text-amber-900 hover:underline"
            >
              Upgrade plan
            </Link>
          </div>
        </NextLink>
      )}
    </>
  );
}
