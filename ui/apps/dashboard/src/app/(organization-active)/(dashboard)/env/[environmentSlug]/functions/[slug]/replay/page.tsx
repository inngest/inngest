'use client';

import React from 'react';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import NewReplayButton from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/NewReplayButton';
import { ReplayList } from './ReplayList';

type FunctionReplayPageProps = {
  params: {
    slug: string;
  };
};
export default function FunctionReplayPage({ params }: FunctionReplayPageProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const env = useEnvironment();

  return (
    <>
      {!env.isArchived && (
        <div className="flex items-center justify-end border-b border-slate-300 px-5 py-2">
          <NewReplayButton functionSlug={functionSlug} />
        </div>
      )}
      <div className="overflow-y-auto">
        <ReplayList functionSlug={functionSlug} />
      </div>
    </>
  );
}
