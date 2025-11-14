'use client';

import { useEffect } from 'react';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';

import { HOME_TAB } from '../InsightsTabManager/constants';
import { guardAgainstActiveQueryIdParamFlash } from './guard';
import { useProcessDeepLink } from './useProcessDeepLink';

type QueryDeepLinkManagerProps = {
  activeSavedQueryId?: string;
  activeTabId: string;
  children: React.ReactNode;
};

export function QueryDeepLinkManager({
  activeSavedQueryId,
  activeTabId,
  children,
}: QueryDeepLinkManagerProps) {
  const [activeQueryIdParam, updateActiveQueryIdParam, removeActiveQueryIdParam] =
    useSearchParam('activeQueryId');

  const { initialDeepLinkIdRef, hasProcessedDeepLinkRef } = useProcessDeepLink(activeQueryIdParam);

  useEffect(() => {
    if (activeSavedQueryId !== undefined) {
      if (activeQueryIdParam !== activeSavedQueryId) updateActiveQueryIdParam(activeSavedQueryId);
      return;
    }

    if (activeQueryIdParam !== undefined) {
      const { shouldGuard } = guardAgainstActiveQueryIdParamFlash({
        activeQueryIdParam,
        hasProcessedDeepLink: hasProcessedDeepLinkRef.current,
        initialDeepLinkId: initialDeepLinkIdRef.current,
        isHomeTabActive: activeTabId === HOME_TAB.id,
      });
      if (shouldGuard) return;

      removeActiveQueryIdParam();
    }
  }, [
    activeQueryIdParam,
    activeSavedQueryId,
    activeTabId,
    hasProcessedDeepLinkRef,
    initialDeepLinkIdRef,
    removeActiveQueryIdParam,
    updateActiveQueryIdParam,
  ]);

  return children;
}
