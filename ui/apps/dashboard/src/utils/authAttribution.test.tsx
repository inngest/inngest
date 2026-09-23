import { createElement, type ComponentType, type ReactNode } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { Route as signInRoute } from '../routes/(auth)/sign-in.$';
import { Route as signUpRoute } from '../routes/(auth)/sign-up.$';

const mocks = vi.hoisted(() => ({
  pathname: '/sign-in',
  signIn: vi.fn(() => null),
  signUp: vi.fn(() => null),
}));

vi.mock('@clerk/tanstack-react-start', () => ({
  SignIn: mocks.signIn,
  SignUp: mocks.signUp,
  ClerkLoaded: ({ children }: { children: ReactNode }) => children,
  ClerkLoading: () => null,
}));

vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => (options: unknown) => ({
    options,
    useSearch: () => ({}),
  }),
  useLocation: () => ({ pathname: mocks.pathname }),
  Link: ({ children }: { children: ReactNode }) => children,
}));

vi.mock('@/components/SignIn/SplitView', () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock('@/components/SignIn/LogoWall', () => ({ default: () => null }));
vi.mock('@/components/SignIn/TrustPanel', () => ({ default: () => null }));
vi.mock('@/utils/urls', () => ({
  canonicalLink: () => ({}),
  absoluteUrl: (path: string) => path,
}));

afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

const attribution = {
  first_utm_source: 'newsletter',
  first_utm_medium: 'email',
  first_utm_campaign: 'launch',
  first_utm_content: 'hero',
  first_utm_term: 'workflows',
  first_landing_url: 'https://www.inngest.com/?utm_source=newsletter',
};
const expectedAttribution = {
  utmSource: 'newsletter',
  utmMedium: 'email',
  utmCampaign: 'launch',
  utmContent: 'hero',
  utmTerm: 'workflows',
  firstLandingUrl: attribution.first_landing_url,
};

// OAuth can create a user on the sign-in callback without visiting sign-up.
describe.each([
  { path: '/sign-in', route: signInRoute, component: mocks.signIn },
  {
    path: '/sign-in/sso-callback',
    route: signInRoute,
    component: mocks.signIn,
  },
  { path: '/sign-up', route: signUpRoute, component: mocks.signUp },
])('Clerk attribution at $path', ({ path, route, component }) => {
  it.each([
    {
      name: 'all first-touch fields and the anonymous ID',
      cookie: `ajs_anonymous_id=anon; inngest_first_touch=${encodeURIComponent(
        JSON.stringify(attribution),
      )}`,
      expected: { anonymousID: 'anon', ...expectedAttribution },
    },
    {
      name: 'attribution without an anonymous ID',
      cookie: `inngest_first_touch=${encodeURIComponent(
        JSON.stringify(attribution),
      )}`,
      expected: expectedAttribution,
    },
    {
      name: 'anonymous ID with a malformed first-touch cookie',
      cookie: 'ajs_anonymous_id=anon; inngest_first_touch=invalid',
      expected: { anonymousID: 'anon' },
    },
    { name: 'no cookies', cookie: '', expected: {} },
    { name: 'server rendering', cookie: undefined, expected: {} },
  ])('passes $name to Clerk', ({ cookie, expected }) => {
    mocks.pathname = path;
    if (cookie !== undefined) vi.stubGlobal('document', { cookie });

    renderToStaticMarkup(
      createElement(route.options.component as ComponentType),
    );

    expect(component).toHaveBeenCalledWith(
      expect.objectContaining({ unsafeMetadata: expected }),
      expect.anything(),
    );
  });
});
