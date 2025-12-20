import { createFileRoute } from '@tanstack/react-router';

const parseDsn = (dsn: string) => {
  const url = new URL(dsn);
  return {
    host: url.host,
    projectId: url.pathname.replace('/', ''),
  };
};

const sentryDsn = process.env.VITE_SENTRY_DSN;
const sentryConfig = sentryDsn ? parseDsn(sentryDsn) : null;

export const Route = createFileRoute('/api/sentry')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        if (!sentryConfig) {
          console.error('VITE_SENTRY_DSN not configured');
          return new Response(null, { status: 200 });
        }

        try {
          const envelope = await request.text();

          const pieces = envelope.split('\n');
          const header = JSON.parse(pieces[0]);

          const incomingDsn = parseDsn(header.dsn || '');

          if (incomingDsn.projectId !== sentryConfig.projectId) {
            console.error(
              'Invalid Sentry project ID:',
              incomingDsn.projectId,
              'expected:',
              sentryConfig.projectId,
            );
            return new Response(null, { status: 200 });
          }

          const sentryUrl = `https://${sentryConfig.host}/api/${sentryConfig.projectId}/envelope/`;

          //
          // Forward relevant headers from the original request
          const forwardHeaders: Record<string, string> = {
            'Content-Type': 'application/x-sentry-envelope',
          };

          const headersToForward = [
            'x-sentry-auth',
            'content-encoding',
            'user-agent',
          ];

          for (const headerName of headersToForward) {
            const value = request.headers.get(headerName);
            if (value) {
              forwardHeaders[headerName] = value;
            }
          }

          const sentryResponse = await fetch(sentryUrl, {
            method: 'POST',
            headers: forwardHeaders,
            body: envelope,
          });

          console.log('Sentry response status:', sentryResponse.status);

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
