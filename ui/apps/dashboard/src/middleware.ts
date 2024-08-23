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
    return authMiddleware().redirectToSignIn({ returnBackUrl: request.url });
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

  return NextResponse.next();
};

export default clerkMiddleware((auth, request) => {
  if (!isPublicRoute(request)) {
    auth().protect();
  }
  return afterAuth(auth, request);
});

export const config = {
  matcher: [
    // Skip Next.js internals and all static files, unless found in search params
    '/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)',
    // Always run for API routes
    '/(api|trpc)(.*)',
  ],
};
