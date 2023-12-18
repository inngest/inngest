'use client';

import { useEnvironment } from '@/queries';
import { Apps } from './Apps';

type Props = {
  params: {
    environmentSlug: string;
  };
};

export default function Page({ params: { environmentSlug } }: Props) {
  const [{ data, error, fetching }] = useEnvironment({ environmentSlug });
  if (error) {
    throw error;
  }
  if (fetching) {
    return null;
  }
  if (!data) {
    // Should be unreachable
    throw new Error('unable to load environment');
  }

  return (
    <div className="overflow-y-auto bg-slate-100">
      <Apps envID={data.id} envSlug={environmentSlug} />
    </div>
  );
}
