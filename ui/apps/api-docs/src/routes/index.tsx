import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/')({
  beforeLoad: () => {
    // Redirect root to docs; /docs matches the /docs/$ catch-all with an empty splat
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    throw redirect({ href: '/docs' } as any);
  },
});
