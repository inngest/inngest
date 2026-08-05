import { useEffect } from 'react';
import { useLocation } from '@tanstack/react-router';

import { useInfoQuery } from '@/store/devApi';
import {
  useFakeDoorTracking,
  type FakeDoorFeature,
} from './useFakeDoorTracking';

const visitedKey = (feature: FakeDoorFeature) =>
  `inngest:fakeDoor:${feature}:visited`;

const dotSeenKey = (feature: FakeDoorFeature) =>
  `inngest:fakeDoor:${feature}:navDotSeen`;

function getFlag(key: string): boolean {
  try {
    return localStorage.getItem(key) === '1';
  } catch {
    return false;
  }
}

function setFlag(key: string) {
  try {
    localStorage.setItem(key, '1');
  } catch {
    // localStorage unavailable; the flag just won't persist.
  }
}

// Called by the Scores/Experiments pages on mount so the nav dot clears
// after the first visit.
export function markFakeDoorVisited(feature: FakeDoorFeature) {
  setFlag(visitedKey(feature));
}

// Unread-dot state for the AI nav items, keyed by href. A dot shows when
// the dev server has detected feature usage (via /dev, fetched once per
// load — no polling) and the user hasn't visited that page yet. Reading
// the router location makes the map recompute on navigation, which is
// what clears the dot after a visit.
export function useFakeDoorNavDots(): Record<string, boolean> {
  const { data: info } = useInfoQuery();
  useLocation();

  const trackScores = useFakeDoorTracking('scores');
  const trackExperiments = useFakeDoorTracking('experiments');

  const scoresDot =
    Boolean(info?.hasSeenScores) && !getFlag(visitedKey('scores'));
  const experimentsDot =
    Boolean(info?.hasSeenExperiments) && !getFlag(visitedKey('experiments'));

  // Track the first time each dot is shown (once per device) so we can
  // measure whether the nav nudge drives fake-door visits.
  useEffect(() => {
    if (scoresDot && !getFlag(dotSeenKey('scores'))) {
      setFlag(dotSeenKey('scores'));
      trackScores('nav_dot_seen');
    }
  }, [scoresDot, trackScores]);
  useEffect(() => {
    if (experimentsDot && !getFlag(dotSeenKey('experiments'))) {
      setFlag(dotSeenKey('experiments'));
      trackExperiments('nav_dot_seen');
    }
  }, [experimentsDot, trackExperiments]);

  return {
    '/ai/scores': scoresDot,
    '/ai/experiments': experimentsDot,
  };
}
