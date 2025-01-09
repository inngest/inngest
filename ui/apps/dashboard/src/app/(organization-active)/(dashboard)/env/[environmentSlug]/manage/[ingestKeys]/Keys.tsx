'use client';

import { notFound } from 'next/navigation';
import { Skeleton } from '@inngest/components/Skeleton';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
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

const LoadingSkeleton = () => (
  <div className="border-subtle border-b px-4 py-3">
    <Skeleton className="mb-1 block h-11 w-full" />
  </div>
);

export default function Keys() {
  const environment = useEnvironment();

  const { data, isLoading, error } = useGraphQLQuery({
    query: GetKeysDocument,
    variables: {
      environmentID: environment.id,
    },
  });

  const keys = data?.environment.ingestKeys;

  function sortFunction(a: { createdAt: string }, b: { createdAt: string }) {
    const dateA = new Date(a.createdAt).getTime();
    const dateB = new Date(b.createdAt).getTime();
    return dateA < dateB ? 1 : -1;
  }

  if (isLoading) {
    return (
      <>
        <LoadingSkeleton />
        <LoadingSkeleton />
      </>
    );
  }

  if (error || !keys) {
    notFound();
  }

  const orderedKeys = keys.sort(sortFunction);

  return (
    <ul role="list" className="h-full overflow-y-auto">
      <KeysListItem list={orderedKeys} />
    </ul>
  );
}
