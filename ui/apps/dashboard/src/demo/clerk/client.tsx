/**
 * Fake `@clerk/tanstack-react-start` for the demo build. Wired in via
 * `resolve.alias` in vite.demo.config.ts so the ~29 client-side Clerk call
 * sites work unchanged against a fixed, always-signed-in demo identity — no
 * real Clerk service, no sign-in required.
 *
 * Only the exports the app actually imports are provided (enumerated from the
 * source). Type-checking (`tsc`) still resolves the real Clerk package, so the
 * job here is runtime shape-compatibility, not type fidelity.
 */
import type { ReactNode } from 'react';

import {
  DEMO_ORG_ID,
  DEMO_SESSION_ID,
  DEMO_TOKEN,
  DEMO_USER_ID,
  demoMembership,
  demoOrg,
  demoSession,
  demoUser,
} from './identity';

// --- Providers / components -------------------------------------------------

export const ClerkProvider = ({ children }: { children: ReactNode }) => (
  <>{children}</>
);

const authPlaceholder = (label: string) => () =>
  (
    <div className="text-muted p-6 text-sm">
      {label} is unavailable in the demo.
    </div>
  );

export const SignIn = authPlaceholder('Sign in');
export const SignUp = authPlaceholder('Sign up');
export const OrganizationList = authPlaceholder('Organization list');
export const OrganizationProfile = authPlaceholder('Organization profile');
export const UserProfile = authPlaceholder('User profile');

// --- Hooks ------------------------------------------------------------------

const noopSignOut = async (cb?: () => void) => {
  cb?.();
};

export const useAuth = () => ({
  isLoaded: true,
  isSignedIn: true,
  userId: DEMO_USER_ID,
  orgId: DEMO_ORG_ID,
  orgRole: 'org:admin',
  sessionId: DEMO_SESSION_ID,
  actor: null,
  has: () => true,
  getToken: async (_opts?: { skipCache?: boolean; template?: string }) =>
    DEMO_TOKEN,
  signOut: noopSignOut,
});

export const useUser = () => ({
  isLoaded: true,
  isSignedIn: true,
  user: demoUser,
});

export const useOrganization = () => ({
  isLoaded: true,
  organization: demoOrg,
  membership: demoMembership,
});

export const useClerk = () => ({
  loaded: true,
  session: demoSession,
  signOut: noopSignOut,
  setActive: async (_opts?: unknown) => {},
  openSignIn: () => {},
  redirectToSignIn: () => {},
});

export const useSignIn = () => ({
  isLoaded: true,
  setActive: async (_opts?: unknown) => {},
  signIn: {
    create: async (_opts?: unknown) => ({ status: 'complete' }),
    attemptFirstFactor: async (_opts?: unknown) => ({ status: 'complete' }),
  },
});
