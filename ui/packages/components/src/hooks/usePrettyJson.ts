import { useMemo } from 'react';

import type { TraceResult } from '../SharedContext/useGetTraceResult';

/**
 * Given a JSON string, returns a pretty-printed version of it if it's valid
 * JSON, else returns `null`.
 */
export const usePrettyJson = (json: string): string | null => {
  return useMemo(() => {
    if (!json) {
      return null;
    }

    try {
      const data: unknown = JSON.parse(json);
      if (data === null) {
        return data;
      }

      return JSON.stringify(data, null, 2);
    } catch (e) {
      console.warn('Unable to parse content as JSON: ', json);
      return '';
    }
  }, [json]);
};

/**
 * Given a serialized error, return a pretty-printed version of the error body.
 * If the error has a `cause`, it will be appended to the end of the body
 * prefixed with `[cause]: `, as it is in usual JS stack traces.
 */
export const usePrettyErrorBody = (error: TraceResult['error'] | undefined): string | null => {
  let cause = '';
  if (typeof error?.cause === 'string') {
    cause = error.cause;
  } else if (error?.cause) {
    cause = JSON.stringify(error.cause);
  }

  // This may be blank as we attempt to parse it as JSON in case it's an object
  // or something we can show nicely.
  const prettyCause = usePrettyJson(cause);

  return useMemo(() => {
    if (!error?.stack) {
      return null;
    }

    let body = error.stack;
    if (error.cause !== null) {
      body += `\n[cause]: ${prettyCause || error.cause || ''}`;
    }
    return body;
  }, [error?.stack, prettyCause]);
};

export const usePrettyShortError = (error: TraceResult['error'] | undefined): string => {
  let cause: string | undefined;
  if (typeof error?.cause === 'string') {
    cause = error.cause;
  } else if (error?.cause) {
    cause = JSON.stringify(error.cause);
  }

  return error?.message
    ? error.message
    : cause
    ? cause
    : error?.name
    ? error.name
    : 'Unknown error';
};
