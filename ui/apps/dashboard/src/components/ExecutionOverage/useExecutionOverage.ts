"use client";

import { useCallback, useEffect, useState } from "react";

import { trackEvent, useTrackingUser } from "@/utils/tracking";
import {
  isFreePlan,
  parseExecutionOverageData,
  useExecutionOverageCheck,
} from "./data";

const STORAGE_KEY = "executionOverageDismissedUntil";

const shouldShowExecutionCTA = (): boolean => {
  if (typeof window === "undefined") return true;

  const until = localStorage.getItem(STORAGE_KEY);
  return !until || new Date(until) < new Date();
};

const dismissExecutionCTA = (hours = 24): void => {
  if (typeof window === "undefined") return;

  const until = new Date();
  until.setHours(until.getHours() + hours);
  localStorage.setItem(STORAGE_KEY, until.toISOString());
};

export function useExecutionOverage() {
  const [isReady, setIsReady] = useState(false);
  const [shouldShow, setShouldShow] = useState(true);
  const trackingUser = useTrackingUser();

  const { data: rawData, usageData, error } = useExecutionOverageCheck();
  const executionOverageData = parseExecutionOverageData(rawData, usageData);

  useEffect(() => {
    setShouldShow(shouldShowExecutionCTA());
    setIsReady(true);
  }, []);

  const dismiss = useCallback(
    (hours = 24) => {
      dismissExecutionCTA(hours);
      setShouldShow(false);

      if (trackingUser) {
        trackEvent({
          name: "app/upsell.execution.overage.dismissed",
          data: {
            variant: "banner",
          },
          user: trackingUser,
          v: "2025-07-14.1",
        });
      }
    },
    [trackingUser],
  );

  const isBannerVisible =
    !error &&
    executionOverageData &&
    executionOverageData.hasExceeded &&
    isFreePlan(executionOverageData.planSlug) &&
    isReady &&
    shouldShow;

  return {
    isBannerVisible,
    executionOverageData,
    error,
    dismiss,
  };
}
