import { createFileRoute } from '@tanstack/react-router';

const SENTRY_HOST = process.env.SENTRY_HOST;
const SENTRY_PROJECT_IDS = process.env.SENTRY_PROJECT_IDS?.split(',') || [];

export const Route = createFileRoute('/api/sentry')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        if (!SENTRY_HOST || SENTRY_PROJECT_IDS.length === 0) {
          console.error('SENTRY_HOST or SENTRY_PROJECT_IDS not configured');
          return new Response(null, { status: 200 });
        }

        try {
          const envelope = await request.text();

          const pieces = envelope.split('\n');
          const header = JSON.parse(pieces[0]);

          const dsn = new URL(header.dsn || '');
          const projectId = dsn.pathname.replace('/', '');

          if (!SENTRY_PROJECT_IDS.includes(projectId)) {
            console.error('Invalid Sentry project ID:', projectId);
            return new Response(null, { status: 200 });
          }

          const sentryUrl = `https://${SENTRY_HOST}/api/${projectId}/envelope/`;

          const sentryResponse = await fetch(sentryUrl, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/x-sentry-envelope',
            },
            body: envelope,
          });

          return new Response(null, {
            status: sentryResponse.status,
          });
        } catch (error) {
          console.error('Error tunneling to Sentry:', error);
          //
          // it's just analytics
          return new Response(null, { status: 200 });
        }
      },
    },
  },
});
