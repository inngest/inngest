type InitialFetchFailed = {
  error: Error;
  data: undefined;
  isLoading: false;
};

type InitialFetchLoading = {
  error: undefined;
  data: undefined;
  isLoading: true;
};

type Skipped = {
  error: undefined;
  data: undefined;
  isLoading: false;
  isSkipped: true;
};

type Succeeded<T = never> = {
  error: undefined;
  data: T;
  isLoading: false;
};

// Same as InitialFetchFailed, but it has data
type RefetchFailed<T = never> = {
  error: Error;
  data: T;
  isLoading: false;
};

// Same as InitialFetchLoading, but it has data
type RefetchLoading<T = never> = {
  error: undefined;
  data: T;
  isLoading: true;
};

export const baseInitialFetchFailed = {
  data: undefined,
  isLoading: false,
  isSkipped: false,
} as const satisfies Omit<InitialFetchFailed, 'error'> & { isSkipped: false };

export const baseInitialFetchLoading = {
  data: undefined,
  error: undefined,
  isLoading: true,
  isSkipped: false,
} as const satisfies InitialFetchLoading & { isSkipped: false };

export const baseFetchSkipped = {
  data: undefined,
  error: undefined,
  isLoading: false,
  isSkipped: true,
} as const satisfies Skipped;

export const baseFetchSucceeded = {
  error: undefined,
  isLoading: false,
  isSkipped: false,
} as const satisfies Omit<Succeeded, 'data'> & { isSkipped: false };

export const baseRefetchFailed = {
  isLoading: false,
  isSkipped: false,
} as const satisfies Omit<RefetchFailed, 'data' | 'error'> & { isSkipped: false };

export const baseRefetchLoading = {
  error: undefined,
  isLoading: true,
  isSkipped: false,
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
> = (
  | InitialFetchFailed
  | InitialFetchLoading
  | Succeeded<TData>
  | RefetchFailed<TData>
  | RefetchLoading<TData>
) & { refetch: () => void };

type Options = {
  skippable?: boolean;
};

export type FetchResult<
  // Required
  TData = never,
  // Optional
  TOptions extends Options = {}
> = TOptions['skippable'] extends true ? FetchResultWithSkip<TData> : FetchResultWithoutSkip<TData>;
