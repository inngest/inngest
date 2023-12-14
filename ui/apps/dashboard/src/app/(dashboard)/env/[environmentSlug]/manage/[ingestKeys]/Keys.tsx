'use client';

import { notFound } from 'next/navigation';
import { Skeleton } from '@inngest/components/Skeleton';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';
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

const LoadingSkeleton = () => (
  <div className="border-b border-slate-100 px-4 py-3">
    <Skeleton className="mb-1 block h-11 w-full" />
  </div>
);

export default function Keys({ environmentSlug }: KeysProps) {
  const [{ data: environment, fetching: fetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });
  const [{ data, fetching: fetchingKey }] = useQuery({
    query: GetKeysDocument,
    variables: {
      environmentID: environment?.id!,
    },
    pause: !environment?.id,
  });

  const loading = fetchingEnvironment || fetchingKey;

  const keys = data?.environment?.ingestKeys;

  function sortFunction(a: { createdAt: string }, b: { createdAt: string }) {
    const dateA = new Date(a.createdAt).getTime();
    const dateB = new Date(b.createdAt).getTime();
    return dateA < dateB ? 1 : -1;
  }

  if (loading) {
    return (
      <>
        <LoadingSkeleton />
        <LoadingSkeleton />
      </>
    );
  }

  if (!keys) {
    notFound();
  }

  const orderedKeys = keys.sort(sortFunction);

  return (
    <ul role="list" className="h-full overflow-y-auto">
      <KeysListItem environmentSlug={environmentSlug} list={orderedKeys} />
    </ul>
  );
}
