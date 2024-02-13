import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import TeamTable from './TeamTable';
import { UserCreator } from './UserCreator';

const Query = graphql(`
  query GetUsers {
    account {
      users {
        createdAt
        email
        id
        lastLoginAt
        name
      }
    }

    session {
      user {
        id
      }
    }
  }
`);

type Props = {};

export default async function Page({}: Props) {
  const res = await graphqlAPI.request(Query);

  const loggedInUserID = res.session?.user.id;

  if (!loggedInUserID) {
    return <div>Not logged in</div>;
  }

  return (
    <div className="flex place-content-center">
      <div>
        <div className="flex">
          <h2 className="mb-4 flex-grow text-lg font-semibold text-gray-900">Team Management</h2>
          <UserCreator />
        </div>

        <TeamTable loggedInUserID={loggedInUserID} users={res.account.users} />
      </div>
    </div>
  );
}
