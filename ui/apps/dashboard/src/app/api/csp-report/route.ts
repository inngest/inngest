const ACCEPTED_TYPES = new Set(['application/csp-report', 'application/reports+json']);
const MAX_BODY_BYTES = 32 * 1024; // 32KB limit

export async function POST(request: Request): Promise<Response> {
  console.log('*** CSP Report Request ***');
  if (!process.env.SENTRY_SECURITY_REPORT_URL) return makeStaticSuccessResponse();
  console.log('Found SENTRY_SECURITY_REPORT_URL:');

  const sentryEnvironment = getSentryEnvironment();
  if (sentryEnvironment === null) return makeStaticSuccessResponse();

  // TODO: Add sentry_release parameter.
  // const reportUrl = `${process.env.SENTRY_SECURITY_REPORT_URL}&sentry_environment=${sentryEnvironment}`;

  const contentType = getMediaType(request.headers.get('content-type'));
  if (contentType === null || !ACCEPTED_TYPES.has(contentType)) {
    return new Response('Unsupported media type', { status: 415 });
  }

  // Consider potentially attacker-controlled; do not parse or evaluate.
  const body = await request.arrayBuffer();

  // Enforce size cap.
  if (body.byteLength > MAX_BODY_BYTES) {
    return new Response('Payload too large', { status: 413 });
  }

  const ctrl = new AbortController();
  const tid = setTimeout(() => ctrl.abort(), 3000);

  try {
    console.log('Would have reported to Sentry.');
    /*
    const res = await fetch(reportUrl, {
      body,
      cache: 'no-store',
      credentials: 'omit',
      headers: { 'content-type': contentType },
      method: 'POST',
      redirect: 'error',
      referrerPolicy: 'no-referrer',
      signal: ctrl.signal,
    });
    console.log('--- Sentry CSP Response ---');
    console.log('Ok:', res.ok);
    console.log('Status:', res.status);
    console.log('End of Sentry CSP Response');
    console.log('--- End Sentry CSP Response ---');
    */
  } catch {
    return new Response(null, { status: 502 });
  } finally {
    clearTimeout(tid);
  }

  return makeStaticSuccessResponse();
}

/**
 * Acceptable content types: "application/csp-report", "application/reports+json".
 * Returns the media type (type/subtype) from a Content-Type header, lowercased,
 * with parameters removed. Example: "Application/CSP-Report; charset=UTF-8" -> "application/csp-report".
 */
function getMediaType(contentTypeHeader: string | null): string | null {
  if (!contentTypeHeader) return null;

  try {
    const base = contentTypeHeader.split(';', 1)[0]?.trim().toLowerCase();
    return base ?? null;
  } catch {
    return null;
  }
}

function makeStaticSuccessResponse(): Response {
  return new Response(null, { status: 204 });
}

// This should match the logic that the @sentry/nextjs package uses.
function getSentryEnvironment(): string | null {
  if (!process.env.VERCEL_ENV) return null;

  return `vercel-${process.env.VERCEL_ENV}`;
}
