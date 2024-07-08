'use client';

import { useCallback } from 'react';
import { maybeDateToString } from '@inngest/components/utils/date';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { useEnvironment } from '../../../environment-context';

const query = graphql(`
  mutation CreateCancellation($input: CreateCancellationInput!) {
    createCancellation(input: $input) {
      id
    }
  }
`);

export function useCreateCancellation({ functionSlug }: { functionSlug: string }) {
  const envID = useEnvironment().id;
  const [, cancelFunction] = useMutation(query);

  return useCallback(
    async ({
      name,
      queuedAtMax,
      queuedAtMin,
    }: {
      name: string | undefined;
      queuedAtMax: Date;
      queuedAtMin: Date | undefined;
    }) => {
      return await cancelFunction({
        input: {
          envID: envID,
          functionSlug,
          name,
          queuedAtMax: queuedAtMax.toISOString(),
          queuedAtMin: maybeDateToString(queuedAtMin),
        },
      });
    },
    [cancelFunction, envID, functionSlug]
  );
}
