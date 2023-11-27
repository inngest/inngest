import { authMiddleware } from '@clerk/nextjs';

export default authMiddleware({
  publicRoutes: ['/password-reset', '/support'],
  ignoredRoutes: '/(images|_next/static|_next/image|favicon)(.*)',
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
