import { useNavigate } from '@tanstack/react-router';
import type { FileRouteTypes } from '@/routeTree.gen';
import { EnvironmentType } from '@/utils/environments';

// Only the standalone demo config substitutes these service hooks. The actual
// OrgMenu, AvatarMenu, ProfileMenu, and SignOutButton components are unchanged.
export function useSkippableGraphQLQuery() {
  return { data: { account: { plan: { name: 'Hobby' } } }, isLoading: false };
}

export default function useOnboardingStep() {
  return { nextStep: null, lastCompletedStep: null };
}

export function useClerk() {
  const navigate = useNavigate();
  return {
    session: { id: 'demo-session' },
    signOut: () => navigate({ to: '/sign-in/choose' as FileRouteTypes['to'] }),
  };
}

export function useEnvironments() {
  return [
    {
      data: [
        {
          slug: 'production',
          name: 'Production',
          type: EnvironmentType.Production,
        },
      ],
    },
  ];
}

export function useQuickSearch({ term }: { term: string; envSlug: string }) {
  const matches = (name: string) =>
    Boolean(term) && name.toLowerCase().includes(term.toLowerCase());
  return {
    data: {
      apps: ['demo-app'].filter(matches).map((name) => ({ name })),
      functions: [{ name: 'Demo function', slug: 'demo-function' }].filter(
        (fn) => matches(fn.name),
      ),
      eventTypes: [{ name: 'demo/event.created' }].filter((event) =>
        matches(event.name),
      ),
      event: null,
      run: null,
    },
    isFetching: false,
    isPending: false,
    error: null,
  };
}
