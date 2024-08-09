'use client';

import { IconApp } from '@inngest/components/icons/App';

import Header from '@/components/Header/old/Header';

export default function Layout({ children }: React.PropsWithChildren) {
  return (
    <>
      <Header icon={<IconApp className="h-5 w-5 text-white" />} title="Unattached Syncs" />
      <div className="h-full overflow-hidden bg-slate-100">{children}</div>
    </>
  );
}
