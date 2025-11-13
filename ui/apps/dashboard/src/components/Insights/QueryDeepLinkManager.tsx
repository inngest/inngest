'use client';

import { useEffect } from 'react';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';

type QueryDeepLinkManagerProps = {
  activeSavedQueryId?: string;
  children: React.ReactNode;
};

export function QueryDeepLinkManager({ activeSavedQueryId, children }: QueryDeepLinkManagerProps) {
  const [currentParamValue, updateActiveQueryIdParam, removeActiveQueryIdParam] =
    useSearchParam('activeQueryId');

  useEffect(() => {
    if (activeSavedQueryId !== undefined) {
      if (currentParamValue !== activeSavedQueryId) updateActiveQueryIdParam(activeSavedQueryId);
      return;
    }

    if (currentParamValue !== undefined) removeActiveQueryIdParam();
  }, [activeSavedQueryId, currentParamValue, removeActiveQueryIdParam, updateActiveQueryIdParam]);

  return children;
}
