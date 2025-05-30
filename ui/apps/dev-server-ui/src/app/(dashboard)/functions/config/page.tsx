'use client';

import { useSearchParams } from 'next/navigation';

import FunctionConfiguration from '@/app/(dashboard)/functions/config/FunctionConfiguration';
import { useGetFunctionQuery } from '@/store/generated';

export default async function FunctionDashboardPage() {
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

  return (
    <div>
      <FunctionConfiguration configuration={data.functionBySlug.configuration} />
    </div>
  );
}
