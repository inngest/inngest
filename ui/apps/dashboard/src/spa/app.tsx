import { RouterProvider, createRouter } from '@tanstack/react-router';

import { routeTree } from './routeTree.gen';

const router = createRouter({
  routeTree,
  basepath: '/tanstack',
});

export default function SPA() {
  return (
    <div className="app">
      <RouterProvider router={router} />
    </div>
  );
}
