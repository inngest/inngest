import type { Route } from 'next';

import { useShared } from './SharedContext';

export type PathCreator = {
  app: (params: { externalAppID: string }) => Route;
  eventPopout: (params: { eventID: string }) => Route;
  eventType?: (params: { eventName: string }) => Route;
  function: (params: { functionSlug: string }) => Route;
  runPopout: (params: { runID: string }) => Route;
  debugger: (params: {
    functionSlug: string;
    runID?: string;
    debugRunID?: string | null;
    debugSessionID?: string | null;
  }) => Route;
};

export const usePathCreator = () => {
  const shared = useShared();
  const pathCreator = shared.pathCreator;

  return {
    pathCreator,
  };
};
