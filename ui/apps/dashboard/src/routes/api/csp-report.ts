import { createFileRoute } from '@tanstack/react-router';
import { inngest } from '@/lib/inngest/client';

export const Route = createFileRoute('/api/csp-report')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        const body = await request.json();
        await inngest.send({
          name: 'app/csp-violation.reported',
          data: body,
        });
        return new Response(null, { status: 200 });
      },
    },
  },
});
