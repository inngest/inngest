'use client';

import { Squares2X2Icon } from '@heroicons/react/20/solid';

import Header from '@/components/Header/Header';

export default function Layout({ children }: React.PropsWithChildren) {
  return (
    <>
      <Header icon={<Squares2X2Icon className="h-5 w-5 text-white" />} title="Unattached Syncs" />
      <div className="h-full overflow-hidden bg-slate-100">{children}</div>
    </>
  );
}
