import { NextResponse } from 'next/server';
import { authMiddleware, redirectToSignIn } from '@clerk/nextjs';

const homepagePath = process.env.NEXT_PUBLIC_HOME_PATH;
if (!homepagePath) {
  throw new Error('The NEXT_PUBLIC_HOME_PATH environment variable is not set');
}

export default authMiddleware({
  publicRoutes: ['/password-reset', '/support', '/api/sentry'],
  ignoredRoutes: '/(images|_next/static|_next/image|favicon)(.*)',
  afterAuth(auth, request) {
    const isSignedIn = !!auth.userId;
    const hasActiveOrganization = !!auth.orgId;
    const isAccountSetup = isSignedIn && hasActiveOrganization && !!auth.sessionClaims.accountID;

    if (!isSignedIn && !auth.isPublicRoute) {
      return redirectToSignIn({ returnBackUrl: request.url });
    }

    if (isSignedIn && !hasActiveOrganization && request.nextUrl.pathname !== '/organization-list') {
      const organizationListURL = new URL('/organization-list', request.url);
      organizationListURL.searchParams.append('redirect_url', request.url);
      return NextResponse.redirect(organizationListURL);
    }

    if (
      isSignedIn &&
      !isAccountSetup &&
      request.nextUrl.pathname !== '/sign-up/account-setup' &&
      request.nextUrl.pathname !== '/organization-list'
    ) {
      return NextResponse.redirect(new URL('/sign-up/account-setup', request.url));
    }

    if (isAccountSetup && request.nextUrl.pathname === '/sign-up/account-setup') {
      return NextResponse.redirect(new URL(homepagePath, request.url));
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
