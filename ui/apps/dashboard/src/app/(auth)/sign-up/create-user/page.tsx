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

export default async function CreateUserPage() {
  await graphqlAPI.request(CreateUserDocument);
  return <ReloadClerkAndRedirect redirectURL="/organization-list" />;
}
