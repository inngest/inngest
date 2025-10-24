import { createRouter } from '@tanstack/react-router';

import { routeTree } from './routeTree.gen';
import { NotFound } from './components/NotFound';
import { Error } from '@inngest/components/Error/Error';
import { queryClient } from './components/StoreProvider';

export const getRouter = () => {
  const router = createRouter({
    routeTree,
    context: { queryClient },
    defaultPreload: 'intent',
    defaultErrorComponent: (err) => (
      <div className="w-full flex my-6">
        <Error message={err.error.message} />
      </div>
    ),
    defaultNotFoundComponent: () => <NotFound />,
  });

  return router;
};
