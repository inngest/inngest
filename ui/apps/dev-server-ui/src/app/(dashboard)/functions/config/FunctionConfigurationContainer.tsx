'use client';

import { FunctionConfiguration } from '@inngest/components/FunctionConfiguration';

import { useGetFunctionQuery, type Function } from '@/store/generated';

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

  // TODO why is as Function needed?
  return (
    <FunctionConfiguration onClose={onClose} inngestFunction={data.functionBySlug as Function} />
  );
}
