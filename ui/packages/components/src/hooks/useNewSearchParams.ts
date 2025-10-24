//
// These hooks provide a generic API for manipulating search params across all routes
// in a Tanstack Router application, similar to Next.js's useSearchParams.
//
// Note: Tanstack Router expects route-specific search schemas for type safety, but these
// hooks work generically across all routes without knowing their specific schemas at
// compile time. We use type assertions on the navigate options to work around TypeScript's
// strict checking while maintaining runtime safety through the hooks' validation logic.
//

import { useCallback, useMemo } from 'react';
import { useNavigate, useSearch, type UseNavigateResult } from '@tanstack/react-router';

import { isStringArray } from '../utils/array';

type SetParam<T> = (value: T) => void;

//
// Helper to navigate with dynamic search params. TypeScript doesn't allow this without
// type assertions because it expects compile-time route schemas, but we need runtime
// flexibility for generic search param manipulation.
//
const navigateWithSearch = (
  navigate: UseNavigateResult<string>,
  updater: (prev: Record<string, unknown>) => Record<string, unknown>
) => {
  navigate({ search: updater as never, replace: true });
};

//
// Returns a tuple of the current value of the search param and a function to
// update it.
//
export const useSearchParam = (
  name: string
): [string | undefined, SetParam<string>, () => void] => {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });

  const upsert = useCallback(
    (value: string) => {
      navigateWithSearch(navigate, (prev) => ({ ...prev, [name]: value }));
    },
    [name, navigate]
  );

  const remove = useCallback(() => {
    navigateWithSearch(navigate, (prev) => {
      const { [name]: _, ...rest } = prev;
      return rest;
    });
  }, [name, navigate]);

  const value = (search as Record<string, unknown>)?.[name];
  return [typeof value === 'string' ? value : undefined, upsert, remove];
};

//
// When we need to manipulate multiple search params at the same time
//
export const useBatchedSearchParams = () => {
  const navigate = useNavigate();

  return useCallback(
    (updates: { [key: string]: string | null }) => {
      navigateWithSearch(navigate, (prev) => {
        const next = { ...prev };
        for (const key in updates) {
          const value = updates[key];
          if (value === null) {
            delete next[key];
          } else {
            next[key] = value;
          }
        }
        return next;
      });
    },
    [navigate]
  );
};

export const useBooleanSearchParam = (
  name: string
): [boolean | undefined, SetParam<boolean>, () => void] => {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });

  const upsert = useCallback(
    (value: boolean) => {
      navigateWithSearch(navigate, (prev) => ({ ...prev, [name]: value ? 'true' : 'false' }));
    },
    [name, navigate]
  );

  const remove = useCallback(() => {
    navigateWithSearch(navigate, (prev) => {
      const { [name]: _, ...rest } = prev;
      return rest;
    });
  }, [name, navigate]);

  const rawValue = (search as Record<string, unknown>)?.[name];
  let value = undefined;
  if (rawValue !== undefined) {
    value = rawValue === 'true';
  }

  return [value, upsert, remove];
};

export const useStringArraySearchParam = (
  name: string
): [Array<string> | undefined, SetParam<Array<string>>, () => void] => {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });

  const upsert = useCallback(
    (value: Array<string>) => {
      navigateWithSearch(navigate, (prev) => ({ ...prev, [name]: JSON.stringify(value) }));
    },
    [name, navigate]
  );

  const remove = useCallback(() => {
    navigateWithSearch(navigate, (prev) => {
      const { [name]: _, ...rest } = prev;
      return rest;
    });
  }, [name, navigate]);

  let value = undefined;
  const rawValue = (search as Record<string, unknown>)?.[name];
  if (typeof rawValue === 'string') {
    try {
      const parsed: unknown = JSON.parse(rawValue);

      if (isStringArray(parsed)) {
        value = parsed;
      } else {
        console.error(`invalid type for search param ${name}`);
      }
    } catch {
      console.error(`invalid JSON for search param ${name}`);
    }
  }

  return [value, upsert, remove];
};

type TypeGuard<T> = (value: any) => value is T;

//
// Use a search param that is validated with a type guard
//
export const useValidatedSearchParam = <T>(
  name: string,
  typeGuard: TypeGuard<T>
): [T | undefined, SetParam<string>] => {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });

  const upsert = useCallback(
    (value: string) => {
      navigateWithSearch(navigate, (prev) => ({ ...prev, [name]: value }));
    },
    [name, navigate]
  );

  const rawValue = (search as Record<string, unknown>)?.[name];
  const value = typeof rawValue === 'string' ? rawValue : undefined;

  if (value === undefined) {
    return [undefined, upsert];
  }

  if (!typeGuard(value)) {
    return [undefined, upsert];
  }

  return [value, upsert];
};

//
// Use a search param that is an array of values that are validated with a type
// guard
//
export const useValidatedArraySearchParam = <T>(
  name: string,
  typeGuard: TypeGuard<T>
): [Array<T> | undefined, SetParam<Array<string>>, () => void] => {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });

  const upsert = useCallback(
    (value: Array<string>) => {
      navigateWithSearch(navigate, (prev) => ({ ...prev, [name]: JSON.stringify(value) }));
    },
    [name, navigate]
  );

  const remove = useCallback(() => {
    navigateWithSearch(navigate, (prev) => {
      const { [name]: _, ...rest } = prev;
      return rest;
    });
  }, [name, navigate]);

  const rawValue = (search as Record<string, unknown>)?.[name];
  const value = useMemo(() => {
    if (rawValue === null || rawValue === undefined) {
      return undefined;
    }

    if (typeof rawValue !== 'string') {
      console.error(`invalid type for search param ${name}: ${rawValue}`);
      return undefined;
    }

    let parsed: unknown;
    try {
      parsed = JSON.parse(rawValue);
    } catch {
      console.error(`invalid JSON for search param ${name}: ${rawValue}`);
      return undefined;
    }

    if (!Array.isArray(parsed)) {
      console.error(`invalid type for search param ${name}: ${rawValue}`);
      return undefined;
    }

    const arr: T[] = [];
    for (const item of parsed) {
      if (typeGuard(item)) {
        arr.push(item);
      } else {
        console.error(`invalid type for search param ${name}: ${item}`);
      }
    }

    return arr;
  }, [name, rawValue, typeGuard]);

  return [value, upsert, remove];
};
