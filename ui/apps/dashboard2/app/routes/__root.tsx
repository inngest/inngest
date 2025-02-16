import type { ReactNode } from 'react';
import { AppRoot } from '@inngest/components/AppRoot';
import { Link, Outlet, createRootRoute } from '@tanstack/react-router';
import { Meta, Scripts } from '@tanstack/start';

export const Route = createRootRoute({
  head: () => ({
    meta: [
      {
        charSet: 'utf-8',
      },
      {
        name: 'viewport',
        content: 'width=device-width, initial-scale=1',
      },
      {
        title: 'TanStack Start Starter',
      },
    ],
  }),
  component: RootComponent,
});

function RootComponent() {
  return (
    <RootDocument>
      <Outlet />
    </RootDocument>
  );
}

function RootDocument({ children }: Readonly<{ children: ReactNode }>) {
  // return (
  //   <html>
  //     <head>
  //       <Meta />
  //     </head>
  //     <body>
  //       <div>
  //         <Link to="/">Home</Link>
  //         <Link to="/env">Env</Link>
  //       </div>
  //       {children}
  //       <Scripts />
  //     </body>
  //   </html>
  // );
  return <AppRoot>{children}</AppRoot>;
}
