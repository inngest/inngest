type FetchFailed = { error: Error; data: undefined; isLoading: false };
// type FetchLoading = { error: undefined; data: undefined; isLoading: true }
type FetchLoading<T = never> = { error: undefined; data: T | undefined; isLoading: true };
type FetchSkipped = { error: undefined; data: undefined; isLoading: false; isSkipped: true };
type FetchSucceeded<T = never> = { error: undefined; data: T; isLoading: false };

export const baseFetchFailed = {
  data: undefined,
  isLoading: false,
  isSkipped: false,
} as const satisfies Omit<FetchFailed, 'error'> & { isSkipped: false };

export const baseFetchLoading = {
  error: undefined,
  isLoading: true,
  isSkipped: false,
} as const satisfies Omit<FetchLoading, 'data'> & { isSkipped: false };

export const baseFetchSkipped = {
  data: undefined,
  error: undefined,
  isLoading: false,
  isSkipped: true,
} as const satisfies FetchSkipped;

export const baseFetchSucceeded = {
  error: undefined,
  isLoading: false,
  isSkipped: false,
} as const satisfies Omit<FetchSucceeded, 'data'> & { isSkipped: false };

// A generic type that represents failed, loading, succeeded, and skipped fetch
// states.
//
// We're using a union because that makes consumption more ergonomic: handling
// error and loading states results in helpful type narrowing.
type FetchResultWithSkip<
  // Required
  TData = never
> = (FetchResultWithoutSkip<TData> & { isSkipped: false }) | FetchSkipped;

// A generic type that represents failed, loading, and succeeded fetch states.
//
// We're using a union because that makes consumption more ergonomic: handling
// error and loading states results in helpful type narrowing.
type FetchResultWithoutSkip<
  // Required
  TData = never
> = FetchFailed | FetchLoading<TData> | FetchSucceeded<TData>;

type Options = {
  skippable?: boolean;
};

export type FetchResult<
  // Required
  TData = never,
  // Optional
  TOptions extends Options = {}
> = TOptions['skippable'] extends true ? FetchResultWithSkip<TData> : FetchResultWithoutSkip<TData>;
