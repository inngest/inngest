import { useCallback } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';

type SetParam = (value: string) => void;

/**
 * Returns a tuple of the current value of the search param and a function to
 * update it.
 */
export function useSearchParam(name: string): [string | undefined, SetParam] {
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
