import type { Route } from 'next';

export const pathCreator = {
  app({ externalAppID }: { externalAppID: string }): Route {
    // TODO: Make this goes to a specific app page when we add that feature
    return '/apps' as Route;
  },
  runPopout({ runID }: { runID: string }): Route {
    return `/runs?runID=${runID}` as Route;
  },
};
