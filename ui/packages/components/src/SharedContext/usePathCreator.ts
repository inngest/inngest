import type { Route } from 'next';

import { useShared } from './SharedContext';

export type PathCreator = {
  aiOverview?: () => Route;
  app: (params: { externalAppID: string }) => Route;
  eventPopout: (params: { eventID: string }) => Route;
  eventType?: (params: { eventName: string }) => Route;
  experiment?: (params: { experimentName: string; functionSlug: string }) => Route;
  function: (params: { functionSlug: string }) => Route;
  runPopout: (params: { runID: string }) => Route;
  // Optional -- not every app rendering Insights has a Sessions feature.
  session?: (params: { sessionKey: string; sessionId: string }) => Route;
  // The session key's own page (every session under that key), as opposed
  // to session()'s one specific (key, id) pair.
  sessions?: (params: { sessionKey: string }) => Route;
};

export const usePathCreator = () => {
  const shared = useShared();
  const pathCreator = shared.pathCreator;

  return {
    pathCreator,
  };
};
