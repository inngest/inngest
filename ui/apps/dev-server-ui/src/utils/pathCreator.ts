import type { Route } from 'next';

export const pathCreator = {
  app({ externalAppID }: { externalAppID: string }): Route {
    // TODO: Make this goes to a specific app page when we add that feature
    return '/apps' as Route;
  },
  function({ functionSlug }: { functionSlug: string }): Route {
    // TODO: Make this goes to a specific app page when we add that feature
    return '/functions' as Route;
  },
  runPopout({ runID }: { runID: string }): Route {
    return `/run?runID=${runID}` as Route;
  },
};
