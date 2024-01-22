import { useEffect, useRef } from 'react';
import {
  baseFetchSkipped,
  baseFetchSucceeded,
  baseInitialFetchFailed,
  baseInitialFetchLoading,
  baseRefetchFailed,
  baseRefetchLoading,
  type FetchResult,
} from '@inngest/components/types/fetch';
import { useQuery, type TypedDocumentNode, type UseQueryArgs } from 'urql';

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
  const [res, executeQuery] = useQuery({
    query,
    variables,
    context,
  });

  useEffect(() => {
    if (res.fetching || !pollIntervalInMilliseconds) {
      return;
    }

    const timeoutID = setTimeout(
      () => executeQuery({ requestPolicy: 'network-only' }),
      pollIntervalInMilliseconds
    );
    return () => clearTimeout(timeoutID);
  }, [res.fetching, pollIntervalInMilliseconds, executeQuery]);

  if (res.fetching) {
    if (!res.data) {
      return baseInitialFetchLoading;
    }

    return {
      ...baseRefetchLoading,
      data: res.data,
    };
  }

  if (res.error) {
    if (!res.data) {
      return {
        ...baseInitialFetchFailed,
        error: new Error(res.error.message),
      };
    }

    return {
      ...baseRefetchFailed,
      data: res.data,
      error: new Error(res.error.message),
    };
  }

  if (!res.data) {
    // Should be unreachable.
    return {
      ...baseInitialFetchFailed,
      error: new Error('finished loading but missing data'),
    };
  }

  return {
    ...baseFetchSucceeded,
    data: res.data,
  };
}

// TODO: Move this function's logic into useGraphQLQuery once we're confident in
// it
export function useGraphQLQuery_TEMPORARY<
  ResultT extends { [key in string]: unknown },
  VariablesT extends { [key in string]: unknown }
>({
  query,
  variables,
  context,
  pollIntervalInMilliseconds,
}: Args<ResultT, VariablesT>): FetchResult<ResultT> {
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
  });

  // Polling hook
  useEffect(() => {
    if (res.fetching || !pollIntervalInMilliseconds) {
      return;
    }

    const timeoutID = setTimeout(
      () => executeQuery({ requestPolicy: 'network-only' }),
      pollIntervalInMilliseconds
    );
    return () => clearTimeout(timeoutID);
  }, [res.fetching, pollIntervalInMilliseconds, executeQuery]);

  if (res.data) {
    dataRef.current = res.data;
  }
  const data = res.data ?? dataRef.current;
  console.log(res);

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
        error: new Error(res.error.message),
      };
    }

    return {
      ...baseRefetchFailed,
      data,
      error: new Error(res.error.message),
    };
  }

  if (!data) {
    // Should be unreachable.
    return {
      ...baseInitialFetchFailed,
      error: new Error('finished loading but missing data'),
    };
  }

  return {
    ...baseFetchSucceeded,
    data,
  };
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
  const [res, executeQuery] = useQuery({
    query,
    variables,
    context,
    pause: skip,
  });

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

  if (res.fetching) {
    if (!res.data) {
      return baseInitialFetchLoading;
    }

    return {
      ...baseRefetchLoading,
      data: res.data,
    };
  }

  if (res.error) {
    if (!res.data) {
      return {
        ...baseInitialFetchFailed,
        error: new Error(res.error.message),
      };
    }

    return {
      ...baseRefetchFailed,
      data: res.data,
      error: new Error(res.error.message),
    };
  }

  if (!res.data) {
    // Should be unreachable.
    return {
      ...baseInitialFetchFailed,
      error: new Error('finished loading but missing data'),
    };
  }

  return {
    ...baseFetchSucceeded,
    data: res.data,
  };
}
