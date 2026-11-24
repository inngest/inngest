// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ProfileDisplayType } from '@/queries/server/profile';
import { Route as OrganizationListRoute } from '@/routes/(auth)/organization-list.$';
import TopBar from '../Layout/TopBar';
import { VerificationPage } from './VerificationPage';
import { initialState } from './demo/scenarios';
import type { VerificationActions } from './types';

vi.mock('@/utils/useGraphQLQuery', async () => import('./demo/shellServices'));
vi.mock(
  '../Onboarding/useOnboardingStep',
  async () => import('./demo/shellServices'),
);
vi.mock('../NavigationV2/Environments', () => ({ default: () => null }));
vi.mock('../Layout/AvatarMenu', () => ({ default: () => null }));
vi.mock('../Layout/SearchTrigger', () => ({ default: () => null }));
vi.mock('../Layout/BottomBar', () => ({ default: () => null }));
vi.mock('@clerk/tanstack-react-start', () => ({
  useAuth: () => ({ isLoaded: true, orgId: 'org-current' }),
  OrganizationList: ({
    afterSelectOrganizationUrl,
  }: {
    afterSelectOrganizationUrl: string;
  }) => <a href={afterSelectOrganizationUrl}>Select organization</a>,
}));

const profile: ProfileDisplayType = {
  isMarketplace: false,
  marketplace: null,
  orgName: 'Current organization',
  displayName: 'Sanjana',
  orgProfilePic: null,
  userProfilePic: null,
};
const actions: VerificationActions = {
  onSendCode: vi.fn(),
  onVerifyCode: vi.fn(),
  onResendCode: vi.fn(),
  onChangeNumber: vi.fn(),
  onSubmitCard: vi.fn(),
  onContinueEmail: vi.fn(),
  onManageEmail: vi.fn(),
  onSwitchCompany: vi.fn(),
  onSwitchProfile: vi.fn(),
  onRetry: vi.fn(),
  onReturn: vi.fn(),
  onContactSupport: vi.fn(),
};

beforeEach(() => {
  window.scrollTo = vi.fn();
});
afterEach(cleanup);

function setup(href: string, organizationRedirectUrl?: string) {
  const root = createRootRoute({ component: Outlet });
  const verification = createRoute({
    getParentRoute: () => root,
    path: '/verify',
    component: () => (
      <VerificationPage
        profile={profile}
        showOnboardingWidget={vi.fn()}
        state={initialState}
        actions={actions}
      />
    ),
  });
  const home = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: () => (
      <TopBar
        profile={profile}
        showOnboardingWidget={vi.fn()}
        organizationRedirectUrl={organizationRedirectUrl}
      />
    ),
  });
  const auth = createRoute({ getParentRoute: () => root, id: '(auth)' });
  const organizations = createRoute({
    getParentRoute: () => auth,
    path: '/organization-list/$',
    component: OrganizationListRoute.options.component,
    validateSearch: OrganizationListRoute.options.validateSearch,
  });
  const router = createRouter({
    routeTree: root.addChildren([
      verification,
      home,
      auth.addChildren([organizations]),
    ]),
    history: createMemoryHistory({ initialEntries: [href] }),
  });
  render(<RouterProvider router={router} />);
  return router;
}

async function switchOrganization() {
  const trigger = await screen.findByRole('button', {
    name: 'C Current organization',
  });
  fireEvent.keyDown(trigger, { key: 'Enter' });
  fireEvent.click(
    await screen.findByRole('menuitem', { name: 'Switch organisations' }),
  );
  return screen.findByRole<HTMLAnchorElement>('link', {
    name: 'Select organization',
  });
}

describe('verification organization switching', () => {
  it.each(['api', 'cli', 'dashboard'])(
    'returns to the complete original %s verification URL after organization selection',
    async (source) => {
      const href = `/verify?account_id=84f0c95b-df57-4624-8354-8622d8aff481&source=${source}&return_to=%2Fenv%2Fproduction%2Fsandboxes#verification`;
      const router = setup(href);
      const destination = await switchOrganization();

      expect(router.state.location.pathname).toBe('/organization-list');
      expect(router.state.location.search).toEqual({ redirect_url: href });
      // The real picker passes this URL to Clerk after the user selects an org.
      expect(destination.getAttribute('href')).toBe(href);
    },
  );

  it('keeps ordinary organization switching on its existing home destination', async () => {
    const router = setup('/');
    const destination = await switchOrganization();

    expect(router.state.location.search.redirect_url).toBeUndefined();
    expect(destination.getAttribute('href')).toBe(
      import.meta.env.VITE_PUBLIC_HOME_PATH || '/',
    );
  });

  it.each(['https://other.example/verify', '//other.example/verify'])(
    'does not forward an external organization redirect: %s',
    async (href) => {
      const router = setup('/', href);
      await switchOrganization();

      expect(router.state.location.search.redirect_url).toBeUndefined();
    },
  );
});
