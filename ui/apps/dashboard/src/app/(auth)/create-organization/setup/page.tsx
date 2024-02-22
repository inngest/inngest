import ReloadClerkAndRedirect from '@/app/(auth)/ReloadClerkAndRedirect';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';

const CreateAccountDocument = graphql(`
  mutation CreateAccount {
    createAccount {
      account {
        id
      }
    }
  }
`);

export default async function OrganizationSetupPage() {
  await graphqlAPI.request(CreateAccountDocument);
  return <ReloadClerkAndRedirect redirectURL="/env/production/apps" />;
}
