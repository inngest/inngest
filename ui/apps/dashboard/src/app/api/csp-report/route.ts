const ACCEPTED_TYPES = new Set(['application/csp-report', 'application/reports+json']);

function getBaseContentType(contentTypeHeader: string | null): string | null {
  if (!contentTypeHeader) return null;

  const base = contentTypeHeader.split(';', 1)[0]?.trim().toLowerCase();
  return base ?? null;
}

export async function POST(request: Request): Promise<Response> {
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

  // Dev-only logging; truncate to avoid noisy output. Never persist.
  if (process.env.NODE_ENV !== 'production') {
    const maxLogChars = 1024;
    const sample = rawText.length > maxLogChars ? rawText.slice(0, maxLogChars) + '…' : rawText;
    // eslint-disable-next-line no-console
    console.info('[csp-report] received', { contentType, sampleLength: sample.length, sample });
  }

  return makeStaticResponse();
}

function makeStaticResponse(): Response {
  return new Response(null, { status: 204 });
}
