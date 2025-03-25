'use client';

import { useCallback, useEffect, useMemo } from 'react';
import type { Route } from 'next';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';

import { isStringArray } from '../utils/array';

type SetParam<T> = (value: T) => void;

/**
 * Returns a tuple of the current value of the search param and a function to
 * update it.
 */
export function useSearchParam(name: string): [string | undefined, SetParam<string>, () => void] {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  const upsert = useCallback(
    (value: string) => {
      const params = new URLSearchParams(searchParams);
      params.set(name, value);

      router.replace((pathname + '?' + params.toString()) as Route);
    },
    [name, pathname, router, searchParams]
  );

  const remove = useCallback(() => {
    const params = new URLSearchParams(searchParams);
    params.delete(name);

    router.replace((pathname + '?' + params.toString()) as Route);
  }, [name, pathname, router, searchParams]);

  const value = searchParams.get(name) ?? undefined;
  return [value, upsert, remove];
}

/**
 * When we need to manipulate multiple search params at the same time
 */
export const useBatchedSearchParams = () => {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  return useCallback(
    (updates: { [key: string]: string | null }) => {
      const params = new URLSearchParams(searchParams);
      Object.entries(updates).forEach(([key, value]) => {
        if (value === null) {
          params.delete(key);
        } else {
          params.set(key, value);
        }
      });
      router.replace((pathname + '?' + params.toString()) as Route);
    },
    [pathname, router, searchParams]
  );
};

export function useBooleanSearchParam(name: string): [boolean | undefined, SetParam<boolean>] {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  const upsert = useCallback(
    (value: boolean) => {
      const params = new URLSearchParams(searchParams);
      params.set(name, value ? 'true' : 'false');

      router.replace((pathname + '?' + params.toString()) as Route);
    },
    [name, pathname, router, searchParams]
  );

  let value = undefined;
  if (searchParams.has(name)) {
    value = searchParams.get(name) === 'true';
  }

  return [value, upsert];
}

export function useStringArraySearchParam(
  name: string
): [Array<string> | undefined, SetParam<Array<string>>, () => void] {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  const upsert = useCallback(
    (value: Array<string>) => {
      const params = new URLSearchParams(searchParams);
      params.set(name, JSON.stringify(value));

      router.replace((pathname + '?' + params.toString()) as Route);
    },
    [name, pathname, router, searchParams]
  );

  const remove = useCallback(() => {
    const params = new URLSearchParams(searchParams);
    params.delete(name);

    router.replace((pathname + '?' + params.toString()) as Route);
  }, [name, pathname, router, searchParams]);

  let value = undefined;
  const rawValue = searchParams.get(name);
  if (typeof rawValue === 'string') {
    const parsed: unknown = JSON.parse(rawValue);

    if (isStringArray(parsed)) {
      value = parsed;
    } else {
      // This means the query param value is the wrong type
      console.error(`invalid type for search param ${name}`);
    }
  }

  return [value, upsert, remove];
}

type TypeGuard<T> = (value: any) => value is T;

/**
 * Use a search param that is validated with a type guard
 */
export function useValidatedSearchParam<T>(
  name: string,
  typeGuard: TypeGuard<T>
): [T | undefined, SetParam<string>] {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  const upsert = useCallback(
    (value: string) => {
      const params = new URLSearchParams(searchParams);
      params.set(name, value);

      router.replace((pathname + '?' + params.toString()) as Route);
    },
    [name, pathname, router, searchParams]
  );

  const value = searchParams.get(name) ?? undefined;
  if (value === undefined) {
    return [undefined, upsert];
  }

  if (!typeGuard(value)) {
    return [undefined, upsert];
  }

  return [value, upsert];
}

/**
 * Use a search param that is an array of values that are validated with a type
 * guard
 */
export function useValidatedArraySearchParam<T>(
  name: string,
  typeGuard: TypeGuard<T>
): [Array<T> | undefined, SetParam<Array<string>>, () => void] {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  const upsert = useCallback(
    (value: Array<string>) => {
      const params = new URLSearchParams(searchParams);
      params.set(name, JSON.stringify(value));

      router.replace((pathname + '?' + params.toString()) as Route);
    },
    [name, pathname, router, searchParams]
  );

  const remove = useCallback(() => {
    const params = new URLSearchParams(searchParams);
    params.delete(name);

    router.replace((pathname + '?' + params.toString()) as Route);
  }, [name, pathname, router, searchParams]);

  const rawValue = searchParams.get(name);
  const value = useMemo(() => {
    if (rawValue === null) {
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
}
