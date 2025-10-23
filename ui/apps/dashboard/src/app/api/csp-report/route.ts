const ACCEPTED_TYPES = new Set(['application/csp-report', 'application/reports+json']);
const MAX_BODY_BYTES = 32 * 1024; // 32KB limit

export async function POST(request: Request): Promise<Response> {
  if (!process.env.SENTRY_SECURITY_REPORT_URL) return makeStaticSuccessResponse();

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
  const _tid = setTimeout(() => ctrl.abort(), 3000);

  /*
  // Forward to Sentry using a clean, credential-free request.
  try {
    await fetch(process.env.SENTRY_SECURITY_REPORT_URL, {
      body,
      cache: 'no-store',
      credentials: 'omit',
      headers: { 'content-type': contentType },
      method: 'POST',
      redirect: 'error',
      referrerPolicy: 'no-referrer',
      signal: ctrl.signal,
    });
  } catch {
    return new Response(null, { status: 502 });
  } finally {
    clearTimeout(tid);
  }
  */

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
