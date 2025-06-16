'use client';

import { FunctionConfiguration } from '@inngest/components/FunctionConfiguration';

import { useGetFunctionQuery } from '@/store/generated';

type FunctionDetailsProps = {
  onClose: () => void;
  functionSlug: string;
};

export function FunctionConfigurationContainer({ onClose, functionSlug }: FunctionDetailsProps) {
  const { data, isFetching } = useGetFunctionQuery(
    { functionSlug: functionSlug },
    {
      refetchOnMountOrArgChange: true,
    }
  );

  if (isFetching || !data || !data.functionBySlug) {
    return null;
  }

  return <FunctionConfiguration onClose={onClose} inngestFunction={data.functionBySlug} />;
}
