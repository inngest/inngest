const ACCEPTED_TYPES = new Set(['application/csp-report', 'application/reports+json']);

function getBaseContentType(contentTypeHeader: string | null): string | null {
  if (!contentTypeHeader) return null;

  const base = contentTypeHeader.split(';', 1)[0]?.trim().toLowerCase();
  return base ?? null;
}

export async function POST(request: Request): Promise<Response> {
  if (process.env.NODE_ENV !== 'production') return makeStaticResponse();

  const contentType = getBaseContentType(request.headers.get('content-type'));
  if (contentType === null || !ACCEPTED_TYPES.has(contentType)) return makeStaticResponse();

  // Read the body and enforce size cap post-read as well.
  let rawText = '';
  try {
    rawText = await request.text();
  } catch {
    // If body can't be read, treat as no-op.
    return makeStaticResponse();
  }

  // TODO: Forward to Sentry.

  return makeStaticResponse();
}

function makeStaticResponse(): Response {
  return new Response(null, { status: 204 });
}
