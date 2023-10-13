type FetchFailed = { error: Error; data: undefined; isLoading: false; isSkipped: false };
type FetchLoading = { error: undefined; data: undefined; isLoading: true; isSkipped: false };
type FetchSkipped = { error: undefined; data: undefined; isLoading: false; isSkipped: true };
type FetchSucceeded<T = never> = { error: undefined; data: T; isLoading: false; isSkipped: false };

// A generic type that represents the possible states of a fetch.
//
// We're using a union because that makes consumption more ergonomic: handling
// error and loading states results in helpful type narrowing.
//
// Default TData to never because it's a required generic.
export type FetchResult<TData = never> =
  | FetchFailed
  | FetchLoading
  | FetchSkipped
  | FetchSucceeded<TData>;
