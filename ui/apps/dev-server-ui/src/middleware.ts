import { NextRequest, NextResponse } from 'next/server';

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow API routes
  if (pathname.startsWith('/api')) {
    return NextResponse.next();
  }

  // Allow login page
  if (pathname === '/login') {
    return NextResponse.next();
  }

  // Check if user is logged in
  const loggedIn = request.cookies.get('inngest_logged_in');

  if (!loggedIn) {
    // Redirect to login
    const loginUrl = new URL('/login', request.url);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    '/((?!api|_next/static|_next/image|favicon.ico).*)',
  ],
};
