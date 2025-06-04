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

  console.log({ data });

  return (
    <div className="grid" style={{ gridTemplateColumns: '1fr 1fr 1fr 432px' }}>
      <div style={{ gridColumn: 'span 3 / span 3' }}></div>
      <div>
        <FunctionConfiguration
          inngestFunction={data.functionBySlug}
          triggers={data.functionBySlug.triggers}
          configuration={data.functionBySlug.configuration}
        />
      </div>
    </div>
  );
}
