import { createMiddleware } from '@tanstack/react-start';
import {
  getResponseHeaders,
  setResponseHeaders,
} from '@tanstack/react-start/server';

export const securityMiddleware = createMiddleware().server(({ next }) => {
  const headers = getResponseHeaders();

  const scriptSrc = [
    'https://analytics-cdn.inngest.com',
    'https://clerk.inngest.com',
    'https://challenges.cloudflare.com',
    'https://unpkg.com',
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
    'https://analytics-cdn.inngest.com',
    'https://analytics.inngest.com',
    'https://inn.gs',
    'https://status.inngest.com',
    'https://clerk.inngest.com',
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
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' https://img.clerk.com",
    "font-src 'self' https://fonts-cdn.inngest.com",
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
