export type GuardArgs = {
  activeQueryIdParam: string | undefined;
  hasProcessedDeepLink: boolean;
  initialDeepLinkId: string | undefined;
  isHomeTabActive: boolean;
};

export type GuardResult = { shouldGuard: boolean };

/**
 * Returns true when we should temporarily avoid removing the activeQueryId param
 * to prevent a visible URL flash while a deep-link is still being processed.
 *
 * Without this, since we land on the home tab, the 'activeQueryId' param would be
 * temporarily removed and then would re-appear once the deep-link opens a new tab.
 *
 * We check a number of conditions to ensure that we don't block legitimate changes
 * to the 'activeQueryId' parameter like opening a different saved query.
 */
export function guardAgainstActiveQueryIdParamFlash(args: GuardArgs): GuardResult {
  const { activeQueryIdParam, hasProcessedDeepLink, initialDeepLinkId, isHomeTabActive } = args;

  if (hasProcessedDeepLink) return { shouldGuard: false };
  if (!isHomeTabActive) return { shouldGuard: false };
  if (activeQueryIdParam !== initialDeepLinkId) return { shouldGuard: false };

  // All guard conditions met; avoid removing the param to prevent URL flash.
  return { shouldGuard: true };
}
