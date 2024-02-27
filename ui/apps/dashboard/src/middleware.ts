import { NextResponse } from 'next/server';
import { authMiddleware, redirectToSignIn } from '@clerk/nextjs';

const homepagePath = process.env.NEXT_PUBLIC_HOME_PATH;
if (!homepagePath) {
  throw new Error('The NEXT_PUBLIC_HOME_PATH environment variable is not set');
}

export default authMiddleware({
  publicRoutes: ['/support', '/api/sentry'],
  ignoredRoutes: '/(images|_next/static|_next/image|favicon)(.*)',
  afterAuth(auth, request) {
    const isSignedIn = !!auth.userId;
    const isUserSetup = isSignedIn && !!auth.sessionClaims.externalID;
    const hasActiveOrganization = !!auth.orgId;
    const isOrganizationSetup = isSignedIn && !!auth.sessionClaims.accountID;

    if (auth.isPublicRoute) {
      return NextResponse.next();
    }

    if (!isSignedIn) {
      return redirectToSignIn({ returnBackUrl: request.url });
    }

    if (!isUserSetup && request.nextUrl.pathname !== '/sign-up/set-up') {
      return NextResponse.redirect(new URL('/sign-up/set-up', request.url));
    }

    if (
      isUserSetup &&
      !hasActiveOrganization &&
      request.nextUrl.pathname !== '/organization-list' &&
      request.nextUrl.pathname !== '/create-organization'
    ) {
      const organizationListURL = new URL('/organization-list', request.url);
      organizationListURL.searchParams.append('redirect_url', request.url);
      return NextResponse.redirect(organizationListURL);
    }

    if (
      isUserSetup &&
      hasActiveOrganization &&
      !isOrganizationSetup &&
      request.nextUrl.pathname !== '/create-organization/set-up'
    ) {
      return NextResponse.redirect(new URL('/create-organization/set-up', request.url));
    }

    return NextResponse.next();
  },
});

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - images (files in public/images)
     * - _next (static files)
     * - _next/image (image optimization files)
     * - favicon (favicon file)
     */
    '/((?!images|_next/static|_next/image|favicon).*)',
  ],
};
