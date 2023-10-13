// A generic type that represents the possible states of a fetch: loading,
// success, and error.
//
// We're using a type union because that makes consumption
// more ergonomic: handling error and loading states results in helpful type
// narrowing.
//
// Default to never because the generic is required.
export type FetchResult<T = never> =
  | { error: undefined; data: undefined; isLoading: true }
  | { error: undefined; data: T; isLoading: false }
  | { error: Error; data: undefined; isLoading: false };
