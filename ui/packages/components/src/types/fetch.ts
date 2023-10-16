type FetchFailed = { error: Error; data: undefined; isLoading: false };
type FetchLoading = { error: undefined; data: undefined; isLoading: true };
type FetchSkipped = { error: undefined; data: undefined; isLoading: false; isSkipped: true };
type FetchSucceeded<T = never> = { error: undefined; data: T; isLoading: false };

// A generic type that represents failed, loading, succeeded, and skipped fetch
// states.
//
// We're using a union because that makes consumption more ergonomic: handling
// error and loading states results in helpful type narrowing.
//
// Default TData to never because it's a required generic.
export type FetchResultWithSkip<TData = never> =
  | (FetchResult<TData> & { isSkipped: false })
  | FetchSkipped;

// A generic type that represents failed, loading, and succeeded fetch states.
//
// We're using a union because that makes consumption more ergonomic: handling
// error and loading states results in helpful type narrowing.
//
// Default TData to never because it's a required generic.
export type FetchResult<TData = never> = FetchFailed | FetchLoading | FetchSucceeded<TData>;
