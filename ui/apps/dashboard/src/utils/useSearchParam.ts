import { useCallback } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';

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

      // @ts-expect-error Router doesn't like strings.
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

      // @ts-expect-error Router doesn't like strings.
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
