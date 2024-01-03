'use client';

import type { PropsWithChildren } from 'react';
import { useSelectedLayoutSegment } from 'next/navigation';

import { SyncList } from './SyncList';

type Props = PropsWithChildren<{
  params: {
    externalID: string;
  };
}>;

export default function Layout({ children, params }: Props) {
  const externalAppID = decodeURIComponent(params.externalID);
  const selectedSyncID = useSelectedLayoutSegment();

  return (
    <div className="flex h-full">
      <SyncList externalAppID={externalAppID} selectedSyncID={selectedSyncID} />
      {children}
    </div>
  );
}
