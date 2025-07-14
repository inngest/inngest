'use client';

import EventsPage from '@/components/Events/EventsPage';
import { useFeatureFlags } from '@/hooks/useFeatureFlags';

export default function Page({}) {
  const { featureFlags } = useFeatureFlags();
  const isEventsEnabled = featureFlags.FEATURE_EVENTS;

  if (!isEventsEnabled) return null;
  return <EventsPage showHeader />;
}
