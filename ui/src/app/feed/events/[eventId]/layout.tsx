'use client';

import { EventSection } from '@/components/Event/Section';

type LayoutProps = {
  children: React.ReactNode;
  params: { eventId: string };
};

export default function Layout({ children, params }: LayoutProps) {
  return (
    <>
      <EventSection eventId={params.eventId} />
      {children}
    </>
  );
}
