import type { VercelCallbackProps } from './callback';
import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/_authed/integrations/vercel/')({
  loader: () => {
    redirect({
      to: '/integrations/vercel/callback',
      search: (prev) => prev as unknown as VercelCallbackProps,
      throw: true,
    });
  },
});
