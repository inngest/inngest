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

// Used in onboarding flow.
const LOCAL_DEV_SERVER_URL = 'http://localhost:8288';

const CLERK_API_URL = 'https://api.clerk.com';
const CLERK_IMG_CDN_URL = 'https://img.clerk.com';
const GOOGLE_TAG_MANAGER_URL = 'https://www.googletagmanager.com';
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

const MONACO_EDITOR_CDN_URL = 'https://cdn.jsdelivr.net/npm/monaco-editor@0.43.0/min/vs';
const MONACO_EDITOR_CDN_SCRIPT_URLS = [
  `${MONACO_EDITOR_CDN_URL}/base/common/worker/simpleWorker.nls.js`,
  `${MONACO_EDITOR_CDN_URL}/base/worker/workerMain.js`,
  `${MONACO_EDITOR_CDN_URL}/basic-languages/javascript/javascript.js`,
  `${MONACO_EDITOR_CDN_URL}/basic-languages/shell/shell.js`,
  // TODO: Add back, this is just for testing.
  // `${MONACO_EDITOR_CDN_URL}/basic-languages/sql/sql.js`,
  `${MONACO_EDITOR_CDN_URL}/editor/editor.main.js`,
  `${MONACO_EDITOR_CDN_URL}/editor/editor.main.nls.js`,
  `${MONACO_EDITOR_CDN_URL}/language/json/jsonMode.js`,
  `${MONACO_EDITOR_CDN_URL}/language/json/jsonWorker.js`,
  `${MONACO_EDITOR_CDN_URL}/language/typescript/tsMode.js`,
  `${MONACO_EDITOR_CDN_URL}/language/typescript/tsWorker.js`,
  `${MONACO_EDITOR_CDN_URL}/loader.js`,
];
const MONACO_EDITOR_CDN_FONT_URL = `${MONACO_EDITOR_CDN_URL}/base/browser/ui/codicons/codicon/codicon.ttf`;
const MONACO_EDITOR_CDN_STYLE_URL = `${MONACO_EDITOR_CDN_URL}/editor/editor.main.css`;

const PROD_URL = 'https://app.inngest.com';

const CSP_REPORT_GROUP = 'csp-endpoint';

// TODO: Add nonce, and remove unsafe-* usages, but that would require dynamic rendering of all pages.
function makeCSPHeader(appURL: string, cspReportURL: string) {
  const isDevBuild = process.env.NODE_ENV === 'development';
  const isProdEnvironment = appURL === PROD_URL;

  const csp = [
    `base-uri 'self'`,
    `connect-src 'self' data: ${LOCAL_DEV_SERVER_URL} ${
      process.env.NEXT_PUBLIC_API_URL ?? ''
    } ${combineCSPURLs(LAUNCHDARKLY_URLS)} ${getClerkURL(
      isProdEnvironment
    )} ${CLERK_API_URL} ${MAZE_PROMPTS_URL} ${INNGEST_STATUS_URL} ${getAllowInnGSURL(
      isProdEnvironment,
      isDevBuild
    )} ${getAllowClerkTelemetryURL(isProdEnvironment)} ${convertUrlToWebSocketURL(
      process.env.NEXT_PUBLIC_API_URL
    )}`,
    `default-src 'self'`,
    `font-src 'self' ${INNGEST_FONT_CDN_URL} ${MONACO_EDITOR_CDN_FONT_URL}`,
    `form-action 'self'`,
    `frame-ancestors 'none'`,
    `frame-src 'self' ${STRIPE_JS_URL} ${getAllowVercelLiveURL(isProdEnvironment, isDevBuild)}`,
    `img-src 'self' data: ${CLERK_IMG_CDN_URL}`,
    `manifest-src 'self'`,
    `object-src 'none'`,
    `script-src 'self' ${combineCSPURLs(MONACO_EDITOR_CDN_SCRIPT_URLS)} ${getClerkURL(
      isProdEnvironment
    )} ${MAZE_SNIPPET_URL} ${INNGEST_UNPKG_CDN_URL} ${STRIPE_JS_URL} ${GOOGLE_TAG_MANAGER_URL} 'wasm-unsafe-eval' 'unsafe-inline' ${getAllowUnsafeEval(
      isDevBuild
    )} ${getAllowVercelLiveURL(isProdEnvironment, isDevBuild)}`,
    `style-src 'self' ${MONACO_EDITOR_CDN_STYLE_URL} 'unsafe-inline'`,
    `worker-src 'self' blob:`,
    // `report-to ${CSP_REPORT_GROUP}`,
    `report-uri ${cspReportURL}`,
  ]
    .map((line) => line.trim())
    .join('; ');

  return csp;
}

const CSP_REPORT_PATH = '/api/csp-report';

function getAppURL(): string | null {
  console.log(1);
  const configuredURL = process.env.NEXT_PUBLIC_APP_URL;
  if (!configuredURL) return null;

  console.log(2);
  if (isValidURL(configuredURL)) return configuredURL;
  console.log(3);

  // Handle preview environments.
  console.log('>>>', configuredURL.includes('$VERCEL_URL'), Boolean(process.env.VERCEL_URL));
  if (configuredURL.includes('$VERCEL_URL') && process.env.VERCEL_URL) {
    console.log('3.1');
    return `https://${process.env.VERCEL_URL}`;
  }
  console.log(4);
  return null;
}

// TODO: Remove -Report-Only once we're confident CSP is working as expected.
function withCSPResponseHeaderReportOnly(response: NextResponse) {
  const appURL = getAppURL();
  if (!appURL) return response;

  let cspReportURL: string | null = null;
  try {
    cspReportURL = new URL(CSP_REPORT_PATH, appURL).toString();
  } catch (_) {
    return response;
  }

  response.headers.set('Content-Security-Policy-Report-Only', makeCSPHeader(appURL, cspReportURL));
  response.headers.set('Reporting-Endpoints', `${CSP_REPORT_GROUP}="${cspReportURL}"`);

  return response;
}

function combineCSPURLs(urls: string[]): string {
  return urls.join(' ');
}

const NON_PROD_CLERK_URL = 'https://saving-seasnail-84.clerk.accounts.dev';
const PROD_CLERK_URL = 'https://clerk.inngest.com';
function getClerkURL(isProdEnvironment: boolean): string {
  return isProdEnvironment ? PROD_CLERK_URL : NON_PROD_CLERK_URL;
}

function getAllowUnsafeEval(isDevBuild: boolean): string {
  return isDevBuild ? "'unsafe-eval'" : '';
}

const VERCEL_LIVE_URL = 'https://vercel.live';
function getAllowVercelLiveURL(isProdEnvironment: boolean, isDevBuild: boolean): string {
  if (isProdEnvironment) return '';
  if (isDevBuild) return '';

  // Preview builds + staging.
  return VERCEL_LIVE_URL;
}

const LOCAL_INN_GS_URL = 'http://127.0.0.1:9999';
const PREVIEW_ENV_INN_GS_URL = 'https://stage.inn.gs';
const PROD_INN_GS_URL = 'https://inn.gs';
function getAllowInnGSURL(isProdEnvironment: boolean, isDevBuild: boolean): string {
  if (isProdEnvironment) return PROD_INN_GS_URL;
  if (isDevBuild) return LOCAL_INN_GS_URL;

  // Preview builds + staging.
  return PREVIEW_ENV_INN_GS_URL;
}

// TODO: Replace with direct env variable.
function convertUrlToWebSocketURL(url: undefined | string): string {
  if (url === undefined) return '';

  try {
    const newUrl = new URL(url);
    newUrl.protocol = newUrl.protocol === 'http:' ? 'ws:' : 'wss:';
    return newUrl.toString();
  } catch (_) {
    return '';
  }
}

const CLERK_TELEMETRY_URL = 'https://clerk-telemetry.com';
function getAllowClerkTelemetryURL(isProdEnvironment: boolean): string {
  return isProdEnvironment ? '' : CLERK_TELEMETRY_URL;
}

function isValidURL(urlString: string): boolean {
  try {
    new URL(urlString);
    return true;
  } catch (_) {
    return false;
  }
}
