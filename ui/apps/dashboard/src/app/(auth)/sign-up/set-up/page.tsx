import ReloadClerkAndRedirect from '@/app/(auth)/ReloadClerkAndRedirect';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';

const CreateUserDocument = graphql(`
  mutation CreateUser {
    createUser {
      user {
        id
      }
    }
  }
`);

export default async function UserSetupPage() {
  await graphqlAPI.request(CreateUserDocument);
  return <ReloadClerkAndRedirect redirectURL="/organization-list" />;
}
