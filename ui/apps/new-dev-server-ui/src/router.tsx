import { createRouter } from '@tanstack/react-router'

import { routeTree } from './routeTree.gen'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { NotFound } from './components/NotFound'
import { Error } from '@inngest/components/Error/Error'

export const getRouter = () => {
  const queryClient = new QueryClient()

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
    Wrap: (props: { children: React.JSX.Element }) => (
      <QueryClientProvider client={queryClient}>
        {props.children}
      </QueryClientProvider>
    ),
  })

  return router
}
