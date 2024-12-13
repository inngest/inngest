'use client';

import { Header } from '@inngest/components/Header/Header';

export default function Layout({ children }: React.PropsWithChildren) {
  return (
    <>
      <Header breadcrumb={[{ text: 'Unattached Syncs', href: '/env' }]} />

      <div className="no-scrollbar h-full overflow-y-scroll">{children}</div>
    </>
  );
}
