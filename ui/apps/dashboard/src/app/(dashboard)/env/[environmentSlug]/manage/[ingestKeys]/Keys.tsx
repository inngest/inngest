import { notFound } from 'next/navigation';

import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import KeysListItem from './KeysListItem';

const GetKeysDocument = graphql(`
  query GetIngestKeys($environmentID: ID!) {
    environment: workspace(id: $environmentID) {
      ingestKeys {
        id
        name
        createdAt
        source
      }
    }
  }
`);

type KeysProps = {
  environmentSlug: string;
};

export default async function Keys({ environmentSlug }: KeysProps) {
  const environment = await getEnvironment({
    environmentSlug,
  });

  const response = await graphqlAPI.request(GetKeysDocument, {
    environmentID: environment.id,
  });

  const keys = response?.environment?.ingestKeys;

  if (!keys) {
    notFound();
  }

  function sortFunction(a: { createdAt: string }, b: { createdAt: string }) {
    const dateA = new Date(a.createdAt).getTime();
    const dateB = new Date(b.createdAt).getTime();
    return dateA < dateB ? 1 : -1;
  }

  const orderedKeys = keys.sort(sortFunction);

  return (
    <ul role="list" className="h-full overflow-y-auto">
      <KeysListItem environmentSlug={environmentSlug} list={orderedKeys} />
    </ul>
  );
}
