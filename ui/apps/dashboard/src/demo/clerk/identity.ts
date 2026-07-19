/**
 * Fixed identity used by the fake Clerk stubs in the demo build. These values
 * are deterministic so the demo always renders the same signed-in user/org and
 * never touches the real Clerk service. Kept separate from the GraphQL mock
 * data because Clerk identity (user/org) and the App API `account` are distinct
 * systems in production; nothing cross-references them at runtime in the demo.
 */

export const DEMO_USER_ID = 'user_demo00000000000000000000';
export const DEMO_ORG_ID = 'org_demo000000000000000000000';
export const DEMO_SESSION_ID = 'sess_demo00000000000000000000';
export const DEMO_ACCOUNT_ID = 'a5e11111-1111-4111-8111-111111111111';
export const DEMO_EXTERNAL_ID = 'demo-external-id';

/** Shape-compatible with the fields the app reads off a Clerk `User`. */
export const demoUser = {
  id: DEMO_USER_ID,
  externalId: DEMO_EXTERNAL_ID,
  firstName: 'Demo',
  lastName: 'User',
  fullName: 'Demo User',
  username: 'demo',
  hasImage: false,
  imageUrl: '',
  primaryEmailAddress: { emailAddress: 'demo@inngest.com' },
  emailAddresses: [{ emailAddress: 'demo@inngest.com' }],
  publicMetadata: {},
};

/** Shape-compatible with the fields the app reads off a Clerk `Organization`. */
export const demoOrg = {
  id: DEMO_ORG_ID,
  name: 'Acme, Inc.',
  slug: 'acme',
  hasImage: false,
  imageUrl: '',
  publicMetadata: { accountID: DEMO_ACCOUNT_ID },
};

/** Shape-compatible with the fields the app reads off an `OrganizationMembership`. */
export const demoMembership = {
  id: 'orgmem_demo0000000000000000000',
  role: 'org:admin',
  organization: demoOrg,
  publicUserData: {
    userId: DEMO_USER_ID,
    firstName: demoUser.firstName,
    lastName: demoUser.lastName,
  },
};

export const demoSession = {
  id: DEMO_SESSION_ID,
  status: 'active',
};

/** A static, obviously-fake bearer token. The mock server ignores auth. */
export const DEMO_TOKEN = 'demo-session-token';
