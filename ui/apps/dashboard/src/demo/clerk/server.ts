/**
 * Fake `@clerk/tanstack-react-start/server` for the demo build. Wired in via
 * `resolve.alias` in vite.demo.config.ts so the server-side Clerk call sites
 * (`auth()`, `clerkClient()`, `clerkMiddleware()`) resolve to a fixed
 * always-authenticated demo identity. Because this reports a fully set-up user
 * with an active, set-up org, `fetchClerkAuth` in src/lib/auth.ts passes all
 * its checks and returns without redirecting to sign-in.
 */
import { createMiddleware } from '@tanstack/react-start';

import {
  DEMO_ORG_ID,
  DEMO_SESSION_ID,
  DEMO_TOKEN,
  DEMO_USER_ID,
  demoMembership,
  demoOrg,
  demoUser,
} from './identity';

export type User = typeof demoUser;
export type Organization = typeof demoOrg;
export type OrganizationMembership = typeof demoMembership;

export const auth = async () => ({
  isAuthenticated: true,
  userId: DEMO_USER_ID,
  orgId: DEMO_ORG_ID,
  orgRole: 'org:admin',
  sessionId: DEMO_SESSION_ID,
  has: () => true,
  getToken: async (_opts?: { template?: string }) => DEMO_TOKEN,
});

export const clerkClient = () => ({
  users: {
    getUser: async (_userId: string) => demoUser,
  },
  organizations: {
    getOrganization: async (_opts: { organizationId: string }) => demoOrg,
    getOrganizationMembershipList: async (_opts: {
      organizationId: string;
    }) => ({
      data: [demoMembership],
      totalCount: 1,
    }),
  },
});

/**
 * Passthrough request middleware mirroring the shape of the real
 * `clerkMiddleware()` so it can sit in the `createStart` requestMiddleware
 * array (see src/start.ts) without populating any real Clerk context.
 */
export const clerkMiddleware = () =>
  createMiddleware().server(({ next }) => next());
