'use client';

import { useSearchParams } from 'next/navigation';
import { FunctionConfiguration } from '@inngest/components/FunctionConfiguration';

import {
  useGetFunctionQuery,
  type Function,
} from '../../../../apps/dev-server-ui/src/store/generated';

type FunctionDetailsProps = {
  onClose: () => void;
};

export function FunctionDetails({ onClose }: FunctionDetailsProps) {
  const params = useSearchParams();

  const functionSlug = params.get('slug');

  if (!functionSlug) return;

  const { data, isFetching } = useGetFunctionQuery(
    { functionSlug: functionSlug },
    {
      refetchOnMountOrArgChange: true,
    }
  );

  if (isFetching || !data || !data.functionBySlug) {
    // TODO Render loading screen
    return null;
  }

  console.log({ data });

  // TODO why is as Function needed?
  return (
    <FunctionConfiguration onClose={onClose} inngestFunction={data.functionBySlug as Function} />
  );
}
