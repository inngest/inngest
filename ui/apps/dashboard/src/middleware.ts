import { NextResponse } from 'next/server';
import {
  clerkMiddleware,
  createRouteMatcher,
  type ClerkMiddlewareAuth,
} from '@clerk/nextjs/server';
import type { NextMiddlewareRequestParam } from 'node_modules/@clerk/nextjs/dist/types/server/types';

const isPublicRoute = createRouteMatcher([
  '/sign-in(.*)',
  '/sign-up(.*)',
  '/support',
  '/api/sentry',
  '/api/inngest(.*)',
]);

const homepagePath = process.env.NEXT_PUBLIC_HOME_PATH;
if (!homepagePath) {
  throw new Error('The NEXT_PUBLIC_HOME_PATH environment variable is not set');
}

const afterAuth = async (
  authMiddleware: ClerkMiddlewareAuth,
  request: NextMiddlewareRequestParam
) => {
  const auth = authMiddleware();
  const isSignedIn = !!auth.userId;
  const isUserSetup = isSignedIn && !!auth.sessionClaims.externalID;
  const hasActiveOrganization = !!auth.orgId;
  const isOrganizationSetup = isSignedIn && !!auth.sessionClaims.accountID;

  if (!isSignedIn) {
    return auth.redirectToSignIn({ returnBackUrl: request.url });
  }

  if (!isUserSetup && request.nextUrl.pathname !== '/sign-up/set-up') {
    return NextResponse.redirect(new URL('/sign-up/set-up', request.url));
  }

  if (
    isUserSetup &&
    !hasActiveOrganization &&
    !request.nextUrl.pathname.startsWith('/create-organization') &&
    !request.nextUrl.pathname.startsWith('/organization-list')
  ) {
    const organizationListURL = new URL('/organization-list', request.url);
    organizationListURL.searchParams.append('redirect_url', request.url);
    return NextResponse.redirect(organizationListURL);
  }

  if (
    isUserSetup &&
    hasActiveOrganization &&
    !isOrganizationSetup &&
    !request.nextUrl.pathname.startsWith('/create-organization') &&
    !request.nextUrl.pathname.startsWith('/organization-list')
  ) {
    return NextResponse.redirect(new URL('/create-organization/set-up', request.url));
  }

  return withCSPResponseHeaderReportOnly(NextResponse.next());
};

export default clerkMiddleware((auth, request) => {
  const hasJwtCookie = request.cookies.getAll().some((cookie) => {
    // Our non-Clerk JWT is either named "jwt" or "jwt-staging".
    return cookie.name.startsWith('jwt');
  });

  if (hasJwtCookie) {
    // Skip Clerk auth for non-Clerk users.
    return withCSPResponseHeaderReportOnly(NextResponse.next());
  }

  // Some clerk-nextjs shenanigans. We must check auth user id before calling
  // auth.protect() below becuase that will always return a 404 by design.
  // https://discord.com/channels/856971667393609759/1021521740800733244/threads/1253004875273338922
  if (!auth().userId && !isPublicRoute(request)) {
    return auth().redirectToSignIn();
  }

  if (!isPublicRoute(request)) {
    auth().protect();
    return afterAuth(auth, request);
  }
});

export const config = {
  matcher: [
    // Skip Next.js internals and all static files, unless found in search params
    '/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)',
  ],
};

const CLERK_IMG_CDN_URL = 'https://img.clerk.com';
const INNGEST_FONT_CDN_URL = 'https://fonts-cdn.inngest.com';
const INNGEST_STATUS_URL = 'https://status.inngest.com';
const INNGEST_UNPKG_CDN_URL = 'https://unpkg.com/@inngest/browser/inngest.min.js';
const MAZE_PROMPTS_URL = 'https://prompts.maze.co';
const MAZE_SNIPPET_URL = 'https://snippet.maze.co';
const STRIPE_JS_URL = 'https://js.stripe.com';

const makeLaunchDarklySubdomainURL = (subdomain: string) => `https://${subdomain}.launchdarkly.com`;
const LAUNCHDARKLY_URLS = [
  makeLaunchDarklySubdomainURL('app'),
  makeLaunchDarklySubdomainURL('clientstream'),
  makeLaunchDarklySubdomainURL('events'),
];

const LOCAL_URLS = ['http://127.0.0.1:8090', 'http://127.0.0.1:9999'];

const MONACO_EDITOR_CDN_URL = 'https://cdn.jsdelivr.net/npm/monaco-editor@0.43.0/min/vs';
const MONACO_EDITOR_CDN_SCRIPT_URLS = [
  `${MONACO_EDITOR_CDN_URL}/base/common/worker/simpleWorker.nls.js`,
  `${MONACO_EDITOR_CDN_URL}/base/worker/workerMain.js`,
  `${MONACO_EDITOR_CDN_URL}/basic-languages/sql/sql.js`,
  `${MONACO_EDITOR_CDN_URL}/editor/editor.main.js`,
  `${MONACO_EDITOR_CDN_URL}/editor/editor.main.nls.js`,
  `${MONACO_EDITOR_CDN_URL}/loader.js`,
];
const MONACO_EDITOR_CDN_FONT_URL = `${MONACO_EDITOR_CDN_URL}/base/browser/ui/codicons/codicon/codicon.ttf`;
const MONACO_EDITOR_CDN_STYLE_URL = `${MONACO_EDITOR_CDN_URL}/editor/editor.main.css`;

// TODO: Add nonce, and remove unsafe-* usages, but that would require dynamic rendering of all pages.
function makeCSPHeader() {
  const isDev = process.env.NODE_ENV === 'development';

  const csp = [
    `base-uri 'self'`,
    `connect-src 'self' ${combineCSPURLs(LAUNCHDARKLY_URLS)} ${getClerkURL(
      isDev
    )} ${MAZE_PROMPTS_URL} ${INNGEST_STATUS_URL} ${combineCSPURLs(LOCAL_URLS)}`,
    `default-src 'self'`,
    `font-src 'self' ${INNGEST_FONT_CDN_URL} ${MONACO_EDITOR_CDN_FONT_URL}`,
    `form-action 'self'`,
    `frame-ancestors 'none'`,
    `frame-src 'self' ${STRIPE_JS_URL}`,
    `img-src 'self' ${CLERK_IMG_CDN_URL}`,
    `manifest-src 'self'`,
    `object-src 'none'`,
    `script-src 'self' ${combineCSPURLs(MONACO_EDITOR_CDN_SCRIPT_URLS)} ${getClerkURL(
      isDev
    )} ${MAZE_SNIPPET_URL} ${INNGEST_UNPKG_CDN_URL} 'unsafe-inline' ${getAllowUnsafeEval(isDev)}`,
    `style-src 'self' ${MONACO_EDITOR_CDN_STYLE_URL} 'unsafe-inline'`,
    `worker-src 'self' blob:`,
  ]
    .map((line) => line.trim())
    .join('; ');

  return csp;
}

// TODO: Remove -Report-Only once we're confident CSP is working as expected.
function withCSPResponseHeaderReportOnly(response: NextResponse) {
  response.headers.set('Content-Security-Policy-Report-Only', makeCSPHeader());
  return response;
}

function combineCSPURLs(urls: string[]): string {
  return urls.join(' ');
}

const NON_PROD_CLERK_URL = 'https://saving-seasnail-84.clerk.accounts.dev';
const PROD_CLERK_URL = 'https://clerk.inngest.com';
function getClerkURL(isDev: boolean): string {
  return isDev ? NON_PROD_CLERK_URL : PROD_CLERK_URL;
}

function getAllowUnsafeEval(isDev: boolean): string {
  return isDev ? "'unsafe-eval'" : '';
}
