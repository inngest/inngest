import ReloadClerkAndRedirect from '@/app/(auth)/ReloadClerkAndRedirect';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { pathCreator } from '@/utils/urls';

const SetUpAccountDocument = graphql(`
  mutation SetUpAccount {
    setUpAccount {
      account {
        id
      }
    }
  }
`);

export default async function OrganizationSetupPage() {
  await graphqlAPI.request(SetUpAccountDocument);
  return <ReloadClerkAndRedirect redirectURL={pathCreator.onboarding()} />;
}
