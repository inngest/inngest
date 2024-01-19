type InitialFetchFailed = {
  error: Error;
  data: undefined;
  isLoading: false;
  status: 'initial_failed';
};

type InitialFetchLoading = {
  error: undefined;
  data: undefined;
  isLoading: true;
  status: 'initial_loading';
};

type Skipped = {
  error: undefined;
  data: undefined;
  isLoading: false;
  isSkipped: true;
  status: 'skipped';
};

type Succeeded<T = never> = {
  error: undefined;
  data: T;
  isLoading: false;
  status: 'succeeded';
};

// Same as InitialFetchFailed, but it has data
type RefetchFailed<T = never> = {
  error: Error;
  data: T;
  isLoading: false;
  status: 'refetch_failed';
};

// Same as InitialFetchLoading, but it has data
type RefetchLoading<T = never> = {
  error: undefined;
  data: T;
  isLoading: true;
  status: 'refetch_loading';
};

export const baseInitialFetchFailed = {
  data: undefined,
  isLoading: false,
  isSkipped: false,
  status: 'initial_failed',
} as const satisfies Omit<InitialFetchFailed, 'error'> & { isSkipped: false };

export const baseInitialFetchLoading = {
  data: undefined,
  error: undefined,
  isLoading: true,
  isSkipped: false,
  status: 'initial_loading',
} as const satisfies InitialFetchLoading & { isSkipped: false };

export const baseFetchSkipped = {
  data: undefined,
  error: undefined,
  isLoading: false,
  isSkipped: true,
  status: 'skipped',
} as const satisfies Skipped;

export const baseFetchSucceeded = {
  error: undefined,
  isLoading: false,
  isSkipped: false,
  status: 'succeeded',
} as const satisfies Omit<Succeeded, 'data'> & { isSkipped: false };

export const baseRefetchFailed = {
  isLoading: false,
  isSkipped: false,
  status: 'refetch_failed',
} as const satisfies Omit<RefetchFailed, 'data' | 'error'> & { isSkipped: false };

export const baseRefetchLoading = {
  error: undefined,
  isLoading: true,
  isSkipped: false,
  status: 'refetch_loading',
} as const satisfies Omit<RefetchLoading, 'data'> & { isSkipped: false };

// A generic type that represents failed, loading, succeeded, and skipped fetch
// states.
//
// We're using a union because that makes consumption more ergonomic: handling
// error and loading states results in helpful type narrowing.
type FetchResultWithSkip<
  // Required
  TData = never
> = (FetchResultWithoutSkip<TData> & { isSkipped: false }) | Skipped;

// A generic type that represents failed, loading, and succeeded fetch states.
//
// We're using a union because that makes consumption more ergonomic: handling
// error and loading states results in helpful type narrowing.
type FetchResultWithoutSkip<
  // Required
  TData = never
> =
  | InitialFetchFailed
  | InitialFetchLoading
  | Succeeded<TData>
  | RefetchFailed<TData>
  | RefetchLoading<TData>;

type Options = {
  skippable?: boolean;
};

export type FetchResult<
  // Required
  TData = never,
  // Optional
  TOptions extends Options = {}
> = TOptions['skippable'] extends true ? FetchResultWithSkip<TData> : FetchResultWithoutSkip<TData>;
