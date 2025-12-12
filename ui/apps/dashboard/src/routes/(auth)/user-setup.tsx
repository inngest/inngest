import ReloadClerkAndRedirect from '@/components/Clerk/ReloadClerkAndRedirect';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { createFileRoute, redirect } from '@tanstack/react-router';
import { createServerFn } from '@tanstack/react-start';

const CreateUserDocument = graphql(`
  mutation CreateUser {
    createUser {
      user {
        id
      }
    }
  }
`);

const createUser = createServerFn({ method: 'GET' }).handler(async () =>
  graphqlAPI.request(CreateUserDocument),
);

export const Route = createFileRoute('/(auth)/user-setup')({
  component: UserSetupComponent,
  beforeLoad: async () => {
    await createUser();
  },
});

function UserSetupComponent() {
  return <ReloadClerkAndRedirect redirectURL="/organization-list" />;
}
