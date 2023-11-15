import {
  baseFetchFailed,
  baseFetchLoading,
  baseFetchSkipped,
  baseFetchSucceeded,
  type FetchResult,
} from '@inngest/components/types/fetch';
import { useQuery, type TypedDocumentNode } from 'urql';

type Args<
  ResultT extends { [key in string]: unknown },
  VariablesT extends { [key in string]: unknown }
> = {
  query: TypedDocumentNode<ResultT, VariablesT>;
  skip: boolean;
  variables: VariablesT;
};

/**
 * Thin wrapper around urql's `useQuery` hook. The purpose is to convert urql's
 * result into a discriminated union, which makes it loading/error/skipped state
 * handling more type safe.
 */
export function useGraphQLQuery<
  ResultT extends { [key in string]: unknown },
  VariablesT extends { [key in string]: unknown }
>({
  query,
  skip,
  variables,
}: Args<ResultT, VariablesT>): FetchResult<ResultT, { skippable: true }> {
  const [res] = useQuery({
    query,
    variables,
    pause: skip,
  });

  if (res.fetching) {
    return baseFetchLoading;
  }

  if (skip) {
    return baseFetchSkipped;
  }

  if (res.error) {
    return {
      ...baseFetchFailed,
      error: new Error(res.error.message),
    };
  }

  if (!res.data) {
    // Should be unreachable.
    return {
      ...baseFetchFailed,
      error: new Error('finished loading but missing data'),
    };
  }

  return {
    ...baseFetchSucceeded,
    data: res.data,
  };
}
