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

const forwardToSentry = async (request: Request) => {
  try {
    if (!sentryConfig?.host || !sentryConfig?.projectId) {
      console.error('VITE_SENTRY_DSN not properly configured');
      return;
    }

    const sentryUrl = `https://${sentryConfig.host}/api/${sentryConfig.projectId}/envelope/`;

    const forwardHeaders = Object.fromEntries(
      ['content-type', 'content-encoding']
        .map((h) => [h, request.headers.get(h)])
        .filter(([, v]) => v),
    );

    const sentryResponse = await fetch(sentryUrl, {
      method: 'POST',
      headers: forwardHeaders,
      body: await request.arrayBuffer(),
    });

    if (sentryResponse.status === 429) {
      console.warn('Sentry rate limit exceeded');
      return;
    }

    if (!sentryResponse.ok) {
      console.warn('Error sending event to Sentry', {
        status: sentryResponse.status,
        statusText: sentryResponse.statusText,
        body: await sentryResponse.json().catch(() => null),
      });
    }
    console.log('debug sentryResponse', sentryResponse);
  } catch (error) {
    console.warn('Error processing Sentry request:', error);
  }
};

export const Route = createFileRoute('/api/sentry')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        await forwardToSentry(request.clone());
        return new Response(null, { status: 200 });
      },
    },
  },
});
