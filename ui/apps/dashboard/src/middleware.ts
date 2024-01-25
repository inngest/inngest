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
    const isAccountSetup =
      isSignedIn && auth.sessionClaims.externalID && auth.sessionClaims.accountID;

    if (!isSignedIn && !auth.isPublicRoute) {
      return redirectToSignIn({ returnBackUrl: request.url });
    }

    if (isSignedIn && !isAccountSetup && request.nextUrl.pathname !== '/sign-up/account-setup') {
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
