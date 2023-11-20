import { authMiddleware } from '@clerk/nextjs';

export default authMiddleware({
  publicRoutes: ['/password-reset', '/support'],
  ignoredRoutes: '/(images|api|_next/static|_next/image|favicon)(.*)',
});

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - images
     * - api (API routes)
     * - _next (static files)
     * - _next/image (image optimization files)
     * - favicon (favicon file)
     */
    '/((?!images|api|_next/static|_next/image|favicon).*)',
  ],
};
