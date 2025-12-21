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
    const envelope = await request.text();

    if (!sentryConfig) {
      console.error('VITE_SENTRY_DSN not configured');
      return;
    }

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
      return;
    }

    const sentryUrl = `https://${sentryConfig.host}/api/${sentryConfig.projectId}/envelope/`;
    const sentryResponse = await fetch(sentryUrl, {
      method: 'POST',
      body: envelope,
    });

    if (sentryResponse.status === 429) {
      console.warn('Sentry rate limit exceeded', sentryResponse);
      return;
    }

    if (!sentryResponse.ok) {
      console.warn(
        'Error sending event to Sentry',
        sentryResponse.statusText,
        sentryResponse,
      );
    }
  } catch (error) {
    console.error('Error processing Sentry request:', error);
  }
};

export const Route = createFileRoute('/api/sentry')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        //
        // it's just analytics, don't block and log if anything goes wrong
        void forwardToSentry(request.clone());
        return new Response(null, { status: 200 });
      },
    },
  },
});
