import { useCallback } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';

import { isStringArray } from '../utils/array';

type SetParam<T> = (value: T) => void;

/**
 * Returns a tuple of the current value of the search param and a function to
 * update it.
 */
export function useSearchParam(name: string): [string | undefined, SetParam<string>] {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  const upsert = useCallback(
    (value: string) => {
      const params = new URLSearchParams(searchParams);
      params.set(name, value);
      router.replace(pathname + '?' + params.toString());
    },
    [name, pathname, router, searchParams]
  );

  const value = searchParams.get(name) ?? undefined;
  return [value, upsert];
}

export function useBooleanSearchParam(name: string): [boolean | undefined, SetParam<boolean>] {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();

  const upsert = useCallback(
    (value: boolean) => {
      const params = new URLSearchParams(searchParams);
      params.set(name, value ? 'true' : 'false');
      router.replace(pathname + '?' + params.toString());
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
      router.replace(pathname + '?' + params.toString());
    },
    [name, pathname, router, searchParams]
  );

  const remove = useCallback(() => {
    const params = new URLSearchParams(searchParams);
    params.delete(name);
    router.replace(pathname + '?' + params.toString());
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
