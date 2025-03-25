// This file contains the utilities for creating and consuming lazy loaded data.
// This is useful for passing around an object that may not be loaded yet, but
// will be loaded in the future. This is useful for components that need to
// fetch data asynchronously, but need to render something in the meantime

const loadingSentinel = Symbol();

// This type is used to represent a value that is lazily loaded; think of it
// like a Promise. It can be either the value itself or a loading symbol. Use
// the isLazyDone type guard to check if the value is loaded
export type Lazy<T> = T | typeof loadingSentinel;

/**
 * Type guard for checking if a lazy loaded value is done loading
 */
export function isLazyDone<T>(data: Lazy<T>): data is T {
  if (data === loadingSentinel) {
    return false;
  }

  return true;
}

/**
 * Convert a nullish value to a lazy loaded value. If the value is nullish then
 * it'll be the loading placeholder, else it'll be the value. Use isLazyDone to
 * type narrow the return value
 */
export function nullishToLazy<T>(data: T | undefined | null): Lazy<T> {
  if (data === undefined || data === null) {
    return loadingSentinel;
  }
  return data;
}
