import { createFileRoute } from '@tanstack/react-router';
import { inngest } from '@/lib/inngest/client';

/**
 * Normalizes CSP report request bodies from two potential structures:
 * 1. { "csp-report": { "blocked-uri": "...", ... } } (kebab-case)
 * 2. { "body": { "blockedURL": "...", ... }, "type": "...", "url": "..." } (camelCase)
 */
function normalizeCspReport(body: unknown): Record<string, unknown> {
  // Handle structure 1: { "csp-report": { ... } }
  if (
    typeof body === 'object' &&
    body !== null &&
    'csp-report' in body &&
    typeof (body as { 'csp-report': unknown })['csp-report'] === 'object' &&
    (body as { 'csp-report': unknown })['csp-report'] !== null
  ) {
    const report = (body as { 'csp-report': Record<string, unknown> })[
      'csp-report'
    ];
    return {
      blockedURL: report['blocked-uri'],
      columnNumber: report['column-number'],
      disposition: report['disposition'],
      documentURL: report['document-uri'],
      effectiveDirective: report['effective-directive'],
      lineNumber: report['line-number'],
      originalPolicy: report['original-policy'],
      referrer: report['referrer'],
      sample: report['sample'],
      sourceFile: report['source-file'],
      statusCode: report['status-code'],
      violatedDirective: report['violated-directive'],
    };
  }

  // Handle structure 2: { "body": { ... }, "type": "...", "url": "..." }
  if (
    typeof body === 'object' &&
    body !== null &&
    'body' in body &&
    typeof (body as { body: unknown }).body === 'object' &&
    (body as { body: unknown }).body !== null
  ) {
    const report = (body as { body: Record<string, unknown> }).body;
    return {
      blockedURL: report['blockedURL'],
      columnNumber: report['columnNumber'],
      disposition: report['disposition'],
      documentURL: report['documentURL'],
      effectiveDirective: report['effectiveDirective'],
      lineNumber: report['lineNumber'],
      originalPolicy: report['originalPolicy'],
      referrer: report['referrer'],
      sample: report['sample'],
      sourceFile: report['sourceFile'],
      statusCode: report['statusCode'],
      violatedDirective: report['violatedDirective'],
    };
  }

  // Fallback: return as-is if structure doesn't match
  return typeof body === 'object' && body !== null
    ? (body as Record<string, unknown>)
    : {};
}

export const Route = createFileRoute('/api/csp-report')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        const body = await request.json();
        const normalizedBody = normalizeCspReport(body);
        await inngest.send({
          name: 'app/csp-violation.reported',
          data: normalizedBody,
        });
        return new Response(null, { status: 200 });
      },
    },
  },
});
