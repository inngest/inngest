import { useEffect, useRef } from 'react';
import { useSearchParams } from 'next/navigation';
import {
  FetchError,
  baseFetchSkipped,
  baseFetchSucceeded,
  baseInitialFetchFailed,
  baseInitialFetchLoading,
  baseRefetchFailed,
  baseRefetchLoading,
  type FetchResult,
} from '@inngest/components/types/fetch';
import { isRecord } from '@inngest/components/utils/typeGuards';
import { CombinedError, useQuery, type TypedDocumentNode, type UseQueryArgs } from 'urql';

import { skipCacheSearchParam } from './urls';

type Args<
  ResultT extends { [key in string]: unknown },
  VariablesT extends { [key in string]: unknown }
> = {
  query: TypedDocumentNode<ResultT, VariablesT>;
  variables: VariablesT;
  context?: UseQueryArgs<VariablesT, ResultT>['context'];
  pollIntervalInMilliseconds?: number;
};

/**
 * Thin wrapper around urql's `useQuery` hook. The purpose is to convert urql's
 * result into a discriminated union, which improves type safety around
 * loading/error/done states.
 */
export function useGraphQLQuery<
  ResultT extends { [key in string]: unknown },
  VariablesT extends { [key in string]: unknown }
>({
  query,
  variables,
  context,
  pollIntervalInMilliseconds,
}: Args<ResultT, VariablesT>): FetchResult<ResultT> {
  // We can reuse `useSkippableGraphQLQuery` because its logic is exactly the
  // same as `useGraphQLQuery`, just with skipping allowed
  const res = useSkippableGraphQLQuery({
    query,
    variables,
    context,
    pollIntervalInMilliseconds,
    skip: false,
  });

  if (res.isSkipped) {
    // Should be unreachable since we hardcoded `skip: false`
    throw new Error();
  }

  return res;
}

/**
 * Thin wrapper around urql's `useQuery` hook. The purpose is to convert urql's
 * result into a discriminated union, which improves type safety around
 * loading/error/skipped/done states.
 */
export function useSkippableGraphQLQuery<
  ResultT extends { [key in string]: unknown },
  VariablesT extends { [key in string]: unknown }
>({
  query,
  skip,
  variables,
  context,
  pollIntervalInMilliseconds,
}: Args<ResultT, VariablesT> & { skip: boolean }): FetchResult<ResultT, { skippable: true }> {
  const searchParams = useSearchParams();
  const skipCache = searchParams.get(skipCacheSearchParam.name) === skipCacheSearchParam.value;

  // Store the result data in a ref because we don't want polling errors to
  // clear that cached data. If urql has a first-class way of doing this then we
  // should use that instead.
  //
  // Use useRef instead of useState because we don't want to trigger a
  // re-render.
  const dataRef = useRef<ResultT | undefined>(undefined);

  const [res, executeQuery] = useQuery({
    query,
    variables,
    context,
    pause: skip,
    requestPolicy: skipCache ? 'network-only' : undefined,
  });

  if (res.data) {
    dataRef.current = res.data;
  }
  const data = res.data ?? dataRef.current;

  // Polling hook
  useEffect(() => {
    if (skip || res.fetching || !pollIntervalInMilliseconds) {
      return;
    }

    const timeoutID = setTimeout(
      () => executeQuery({ requestPolicy: 'network-only' }),
      pollIntervalInMilliseconds
    );
    return () => clearTimeout(timeoutID);
  }, [skip, res.fetching, pollIntervalInMilliseconds, executeQuery]);

  if (skip) {
    return baseFetchSkipped;
  }

  // Handle both fetching states (initial fetch and refetch)
  if (res.fetching) {
    if (!data) {
      return baseInitialFetchLoading;
    }

    return {
      ...baseRefetchLoading,
      data,
    };
  }

  // Handle both error states (initial fetch and refetch)
  if (res.error) {
    if (!data) {
      return {
        ...baseInitialFetchFailed,
        error: toFetchError(res.error),
      };
    }

    return {
      ...baseRefetchFailed,
      data,
      error: toFetchError(res.error),
    };
  }

  if (!data) {
    // Should be unreachable.
    return {
      ...baseInitialFetchFailed,
      error: new FetchError('finished loading but missing data'),
    };
  }

  return {
    ...baseFetchSucceeded,
    data,
  };
}

function toFetchError(error: CombinedError): FetchError {
  let code;
  let data;
  for (const graphQLError of error.graphQLErrors) {
    if (graphQLError.extensions.code && typeof graphQLError.extensions.code === 'string') {
      // Use the first valid error code we find. Technically there could be
      // multiple error codes, but that's a complexity we don't need to handle
      // right now
      code = graphQLError.extensions.code;

      if (isRecord(graphQLError.extensions.data)) {
        data = graphQLError.extensions.data;
      }

      break;
    }
  }

  return new FetchError(error.message, {
    code: code ?? 'unknown',
    data,
  });
}
