import ReloadClerkAndRedirect from '@/components/Clerk/ReloadClerkAndRedirect';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import {
  ACCOUNT_CREATED_TIMEOUT_MS,
  trackAccountCreated,
} from '@/utils/accountCreatedTracking';
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

const ClaimSignupConversionDocument = graphql(`
  mutation ClaimSignupConversion($accountID: UUID!) {
    claimSignupConversion(accountID: $accountID)
  }
`);

const claimSignupConversion = createServerFn({ method: 'POST' })
  .inputValidator((accountID: string) => accountID)
  .handler(async ({ data }) => {
    const { getRequest } = await import('@tanstack/react-start/server');
    const result = await graphqlAPI.request({
      document: ClaimSignupConversionDocument,
      variables: { accountID: data },
      signal: AbortSignal.any([
        getRequest().signal,
        AbortSignal.timeout(ACCOUNT_CREATED_TIMEOUT_MS),
      ]),
    });
    return result.claimSignupConversion;
  });

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
          claimConversion: (signal) =>
            setUpAccountID
              ? claimSignupConversion({ data: setUpAccountID, signal })
              : Promise.resolve(false),
        })
      }
    />
  );
}
