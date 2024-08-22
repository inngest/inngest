'use client';

import { Header } from '@inngest/components/Header/Header';

export default function Layout({ children }: React.PropsWithChildren) {
  return (
    <>
      <Header breadcrumb={[{ text: 'Unattached Syncs', href: '/env' }]} />

      <div className="no-scrollbar overflow-y-scroll p-6">{children}</div>
    </>
  );
}
