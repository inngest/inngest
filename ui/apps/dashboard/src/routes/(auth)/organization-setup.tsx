import ReloadClerkAndRedirect from '@/components/Clerk/ReloadClerkAndRedirect';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { trackAccountCreated } from '@/utils/accountCreatedTracking';
import { canonicalLink, pathCreator } from '@/utils/urls';
import { createFileRoute } from '@tanstack/react-router';
import { createServerFn } from '@tanstack/react-start';

const SetUpAccountDocument = graphql(`
  mutation SetUpAccount {
    setUpAccount {
      account {
        id
      }
    }
  }
`);

const setUpAccount = createServerFn({ method: 'GET' }).handler(() =>
  graphqlAPI.request(SetUpAccountDocument),
);

export const Route = createFileRoute('/(auth)/organization-setup')({
  component: OrganizationSetupComponent,
  head: () => ({
    links: [canonicalLink('/organization-setup')],
    meta: [{ name: 'robots', content: 'noindex' }],
  }),
  beforeLoad: async () => {
    const result = await setUpAccount();
    return { setUpAccountID: result.setUpAccount?.account?.id ?? null };
  },
});

function OrganizationSetupComponent() {
  const { setUpAccountID } = Route.useRouteContext();

  return (
    <ReloadClerkAndRedirect
      redirectURL={pathCreator.onboarding()}
      beforeRedirect={(user) =>
        trackAccountCreated({
          accountID: setUpAccountID,
          email: user.primaryEmailAddress?.emailAddress,
          isFirstOrganization: user.organizationMemberships.length === 1,
        })
      }
    />
  );
}
