import { createMiddleware } from '@tanstack/react-start';
import {
  getResponseHeaders,
  setResponseHeaders,
} from '@tanstack/react-start/server';

export const securityMiddleware = createMiddleware().server(({ next }) => {
  const headers = getResponseHeaders();

  const monacoEditorCdn = 'https://cdn.jsdelivr.net';

  const scriptSrc = [
    'https://analytics-cdn.inngest.com',
    'https://clerk.inngest.com',
    'https://challenges.cloudflare.com',
    'https://unpkg.com/@inngest/', // Inngest browser SDK
    monacoEditorCdn,
  ];
  if (
    process.env.VERCEL_ENV === 'preview' ||
    process.env.NODE_ENV === 'development'
  ) {
    scriptSrc.push('https://*.clerk.accounts.dev');
  }
  if (process.env.NODE_ENV === 'development') {
    scriptSrc.push('https://cdn.segment.com');
  }
  const connectSrc = [
    process.env.VITE_API_URL, // e.g. https://api.inngest.com
    process.env.VITE_API_URL?.replace(/$https/, 'wss'),
    'https://analytics-cdn.inngest.com',
    'https://analytics.inngest.com',
    'https://inn.gs',
    process.env.VITE_EVENT_API_HOST,
    'https://status.inngest.com',
    'https://clerk.inngest.com',
    'https://localhost:8288', // Direct communication with the dev server
    'https://clientstream.launchdarkly.com',
    'https://events.launchdarkly.com',
    'https://app.launchdarkly.com',
  ];
  if (
    process.env.VERCEL_ENV === 'preview' ||
    process.env.NODE_ENV === 'development'
  ) {
    connectSrc.push('https://*.clerk.accounts.dev', 'https://vercel.live');
  }
  if (process.env.NODE_ENV === 'development') {
    connectSrc.push('https://cdn.segment.com', 'https://api.segment.io');
  }

  const csp = [
    "default-src 'self'",
    `script-src 'self' 'unsafe-inline' ${scriptSrc.join(' ')}`,
    `connect-src 'self' ${connectSrc.join(' ')}`,
    `style-src 'self' 'unsafe-inline' ${monacoEditorCdn}`, // Monaco editor
    "img-src 'self' data: https://img.clerk.com",
    `font-src 'self' https://fonts-cdn.inngest.com ${monacoEditorCdn}`,
    "frame-src 'self' https://challenges.cloudflare.com",
    "worker-src 'self' blob:",
    "base-uri 'self'",
    "form-action 'self'",
    process.env.VITE_MODE === 'production' && 'upgrade-insecure-requests',
    'report-uri /api/csp-report; report-to csp-endpoint',
  ]
    .filter(Boolean)
    .join('; ');

  // Replace newline characters and spaces
  const contentSecurityPolicyHeaderValue = csp.replace(/\s{2,}/g, ' ').trim();

  // TODO: Uncomment this when we are ready to enforce the CSP
  // headers.set('Content-Security-Policy', contentSecurityPolicyHeaderValue);
  headers.set(
    'Content-Security-Policy-Report-Only',
    contentSecurityPolicyHeaderValue,
  );
  headers.set('Reporting-Endpoints', 'csp-endpoint="/api/csp-report"');
  headers.set('X-Frame-Options', 'DENY');
  headers.set('X-Content-Type-Options', 'nosniff');
  headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');

  setResponseHeaders(headers);

  return next();
});
