"use client";

import Link from "next/link";
import { Button } from "@inngest/components/Button";
import { MenuItem } from "@inngest/components/Menu/MenuItem";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@inngest/components/Tooltip/Tooltip";
import { RiCloseLine, RiErrorWarningFill } from "@remixicon/react";

import { pathCreator } from "@/utils/urls";
import { useSeatOverage } from "./useSeatOverage";

// TODO: turn into a component for all other upsell widgets
export default function SeatOverageWidget({
  collapsed,
}: {
  collapsed: boolean;
}) {
  const { isWidgetVisible, seatOverageData, dismiss } = useSeatOverage();

  // Track CTA viewed when widget becomes visible (temporarily disabled)
  // useEffect(() => {
  //   if (isWidgetVisible && seatOverageData && trackingUser) {
  //     trackEvent({
  //       name: 'app/billing.cta.viewed',
  //       data: {
  //         cta: collapsed ? 'seat-overage-widget-collapsed' : 'seat-overage-widget-expanded',
  //         entitlement: 'user_seats',
  //       },
  //       user: trackingUser,
  //       v: '2025-01-15.1',
  //     });
  //   }
  // }, [isWidgetVisible, collapsed, seatOverageData, trackingUser]);

  if (!isWidgetVisible || !seatOverageData) {
    return null;
  }

  return (
    <>
      {collapsed && (
        <MenuItem
          href={pathCreator.billing({
            tab: "plans",
            ref: "seat-overage-widget-collapsed",
          })}
          className="border border-amber-200 bg-amber-50"
          collapsed={collapsed}
          text="Upgrade plan"
          icon={
            <RiErrorWarningFill className="h-[18px] w-[18px] text-amber-600" />
          }
        />
      )}

      {!collapsed && (
        <div className="text-basis mb-5 block rounded border border-amber-200 bg-amber-50 p-3 leading-tight">
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
                        dismiss();
                      }}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="right" className="max-w-40">
                    <p>Dismiss for 24 hours</p>
                  </TooltipContent>
                </Tooltip>
              </div>
              <p className="flex items-center gap-1.5 text-amber-800">
                Seat limit exceeded
              </p>
              <p className="text-sm text-amber-700">
                You&apos;re using {seatOverageData.userCount} seats but your
                plan includes {seatOverageData.userLimit}.
              </p>
            </div>
            <Link
              href={pathCreator.billing({
                ref: "seat-overage-widget-expanded",
              })}
              className="text-sm text-amber-800 hover:text-amber-900 hover:underline"
            >
              Upgrade plan
            </Link>
          </div>
        </div>
      )}
    </>
  );
}
