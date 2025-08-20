import { createRouter } from '@tanstack/react-router';

// Shared router creation logic that can be used by both main.tsx and Next.js
export async function createTanStackRouter(basepath?: string) {
  try {
    console.log('Creating TanStack Router with basepath:', basepath);

    // Import routes dynamically (using full relative paths from src)
    console.log('Importing routes...');
    const [rootModule, indexModule, aboutModule] = await Promise.all([
      import('./routes/__root'),
      import('./routes/index'),
      import('./routes/about'),
    ]);

    console.log('Routes imported successfully:', { rootModule, indexModule, aboutModule });

    // Build the route tree
    console.log('Building route tree...');
    const routeTree = rootModule.Route.addChildren([indexModule.Route, aboutModule.Route]);
    console.log('Route tree built successfully');

    // Create router with optional basepath
    console.log('Creating router instance...');
    const router = createRouter({
      routeTree,
      ...(basepath && { basepath }),
    });

    console.log('Router created successfully');
    return router;
  } catch (error) {
    console.error('Error in createTanStackRouter:', error);
    throw error;
  }
}

// Type registration for the router
declare module '@tanstack/react-router' {
  interface Register {
    router: Awaited<ReturnType<typeof createTanStackRouter>>;
  }
}
