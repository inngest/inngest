'use client';

import { notFound } from 'next/navigation';
import { ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { Skeleton } from '@inngest/components/Skeleton';

import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';
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

type KeysProps = {
  environmentSlug: string;
};

const LoadingSkeleton = () => (
  <div className="border-b border-slate-100 px-4 py-3">
    <Skeleton className="mb-1 block h-11 w-full" />
  </div>
);

export default function Keys({ environmentSlug }: KeysProps) {
  const [{ data: environment, fetching: fetchingEnvironment, error: environmentError }] =
    useEnvironment({
      environmentSlug,
    });

  const { data, isLoading, error } = useGraphQLQuery({
    query: GetKeysDocument,
    variables: {
      environmentID: environment?.id || '',
    },
    skip: !environment?.id,
  });

  const loading = fetchingEnvironment || isLoading;

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

  if (environmentError || error) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-5">
        <div className="inline-flex items-center gap-2 text-red-600">
          <ExclamationCircleIcon className="h-4 w-4" />
          <h2 className="text-sm">{`Could not load ${
            environmentError ? 'environment' : 'list'
          }`}</h2>
        </div>
      </div>
    );
  }

  const orderedKeys = keys.sort(sortFunction);

  return (
    <ul role="list" className="h-full overflow-y-auto">
      <KeysListItem environmentSlug={environmentSlug} list={orderedKeys} />
    </ul>
  );
}
