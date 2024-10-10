'use client';

import { useMemo } from 'react';
import z from 'zod';

import { CodeBlock } from './CodeBlock';

type Props = {
  className?: string;
  result: string;
  isSuccess: boolean;
};

export function RunResult({ className, result, isSuccess }: Props) {
  const parsedResult = useParsedResult({ isSuccess, result });

  return (
    <div className={className}>
      {parsedResult.isSuccess && (
        <CodeBlock
          header={{
            title: 'Output',
            status: 'success',
          }}
          tab={{
            content: parsedResult.data,
          }}
        />
      )}

      {!parsedResult.isSuccess && (
        <CodeBlock
          header={{
            title: `${parsedResult.error.name}: ${parsedResult.error.message}`,
            status: 'error',
          }}
          tab={{
            content: parsedResult.error.stack,
          }}
        />
      )}
    </div>
  );
}

const errorSchema = z.object({
  name: z.string().nullish(),
  message: z.string(),
  stack: z.string().nullish(),
});

function useParsedResult({ isSuccess, result }: { isSuccess: boolean; result: string }) {
  return useMemo(() => {
    if (isSuccess) {
      let data: unknown;
      try {
        data = JSON.parse(result);
      } catch {
        return {
          data: result,
          isSuccess,
        };
      }
      return {
        data: JSON.stringify(data, null, 2),
        isSuccess,
      };
    }

    const parsedError = errorSchema.safeParse(JSON.parse(result));
    if (!parsedError.success) {
      return {
        error: {
          name: 'Error',
          message: '',
          stack: result,
        },
        isSuccess,
      };
    }

    return {
      error: {
        ...parsedError.data,
        name: parsedError.data.name ?? 'Error',
        stack: parsedError.data.stack ?? '',
      },
      isSuccess,
    };
  }, [isSuccess, result]);
}
