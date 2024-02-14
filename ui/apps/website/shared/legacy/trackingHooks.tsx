import deterministicSplit from "deterministic-split";
import { useEffect, useMemo } from "react";
import { useCookie, useLocalStorage } from "react-use";
import { v4 as uuid } from "uuid";

/**
 * AB experiments with keys as experiment names and values as the variants.
 */
export const abExperiments = {
  // e.g. "2022-01-01-experiment-name": ["variant-1", "variant-2"]
} as const;

/**
 * Fetch and return the user's anonymous ID.
 */
export const useAnonymousID = (): { anonymousID: string; existing: boolean } => {
  const [anonymousID, setAnonymousID] = useCookie("inngest_anonymous_id");

  // TODO: remove this once sufficient time has passed for users to get the new cookie.
  // If the user has a legacy anonymous ID, migrate it to the new cookie.
  const [legacyAnonymousID, _, deleteLegacyAnonymousID] = useLocalStorage<string>("inngest-anon-id");
  if (!anonymousID && legacyAnonymousID) {
    const sixMonthsFromNow = new Date(Date.now() + 6 * 30 * 24 * 60 * 60 * 1000);
    setAnonymousID(legacyAnonymousID, { domain: process.env.NEXT_PUBLIC_HOSTNAME, path: "/", expires: sixMonthsFromNow, sameSite: 'lax' });
    deleteLegacyAnonymousID();
  }

  if (!anonymousID) {
    const newAnonymousID = uuid();
    const sixMonthsFromNow = new Date(Date.now() + 6 * 30 * 24 * 60 * 60 * 1000);
    setAnonymousID(newAnonymousID, { domain: process.env.NEXT_PUBLIC_HOSTNAME, path: "/", expires: sixMonthsFromNow, sameSite: 'lax' });
    return {
      anonymousID: newAnonymousID,
      existing: false,
    };
  }

  return {
    anonymousID,
    existing: true,
  };
};

// If we call useAbTest in multiple places in one page,
// we should only track it once per experiment.
const tracked: { [key: string]: boolean } = Object.keys(abExperiments).reduce(
  (acc, key) => {
    return { ...acc, [key]: false };
  },
  {}
);

/**
 * Fetch the user's current variant for a particular AB experiment.
 */
export const useAbTest = <T extends keyof typeof abExperiments>(
  /**
   * The experiment name to return the variant of.
   */
  experimentName: T
) => {
  const { anonymousID } = useAnonymousID();

  const variant = useMemo(() => {
    // for server side rendering, always render the first variant
    if (typeof window === "undefined") {
      return abExperiments[experimentName][0];
    }
    return deterministicSplit(
      `${anonymousID}_${experimentName}`,
      abExperiments[experimentName]
    ) as typeof abExperiments[T][number];
  }, [anonymousID, experimentName]);

  /**
   * Whenever the variant and fetched and used, send an event to mark that this
   * has happened.
   */
  useEffect(() => {
    // Inngest is undefined on server side during local dev
    if (window?.Inngest && !tracked[experimentName]) {
      window.Inngest.event({
        name: "website/experiment.viewed",
        data: {
          anonymous_id: anonymousID,
          experiment: experimentName,
          variant,
        },
      });
      tracked[experimentName] = true;
    }
  }, [variant, anonymousID, experimentName]);

  return {
    /**
     * The variant of the given experiment for this user.
     */
    variant,
  };
};
