'use client';

import { use } from 'react';
import dynamic from 'next/dynamic';

import EventsPage from '@/components/Events/EventsPage';

const EventsFeedback = dynamic(() => import('@/components/Surveys/EventsFeedback'), {
  ssr: false,
});

export default function Page(props: { params: Promise<{ environmentSlug: string }> }) {
  const params = use(props.params);

  const { environmentSlug: envSlug } = params;

  return (
    <>
      <EventsPage environmentSlug={envSlug} showHeader />
      <EventsFeedback />
    </>
  );
}
