'use client';

import EventsPage from '@/components/Events/EventsPage';

export default function Page({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  return <EventsPage environmentSlug={envSlug} showHeader />;
}
