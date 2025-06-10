'use client';

import { useSearchParams } from 'next/navigation';
import { FunctionConfiguration } from '@inngest/components/FunctionConfiguration';

import { useGetFunctionQuery } from '../../../../apps/dev-server-ui/src/store/generated';

type FunctionDetailsProps = {
  onClose: () => void;
};

export function FunctionDetails({ onClose }: FunctionDetailsProps) {
  const params = useSearchParams();

  const functionSlug = params.get('slug');

  const { data, isFetching } = useGetFunctionQuery(
    { functionSlug: functionSlug },
    {
      refetchOnMountOrArgChange: true,
    }
  );

  if (isFetching) {
    // TODO Render loading screen
    return null;
  }

  console.log({ data });

  return (
    <FunctionConfiguration
      onClose={onClose}
      inngestFunction={data.functionBySlug}
      triggers={data.functionBySlug.triggers}
      configuration={data.functionBySlug.configuration}
    />
  );
}
