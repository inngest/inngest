import { useCallback, useEffect, useMemo, useState } from 'react';

import { trackEvent, useTrackingUser } from '@/utils/tracking';
import {
  announcements as allAnnouncements,
  type Announcement,
} from './announcements';

const STORAGE_KEY = 'dismissedAnnouncements';
// Session-scoped so a viewed event fires at most once per id per browser
// session, surviving SPA remounts but resetting on a genuine new session.
const VIEWED_STORAGE_KEY = 'viewedAnnouncements';
// Bump when the shape/meaning of the announcement tracking events changes.
const TRACK_VERSION = '2026-06-18.1';

function readDismissed(): string[] {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((v): v is string => typeof v === 'string');
  } catch (error) {
    console.warn(`error reading localStorage key "${STORAGE_KEY}":`, error);
    return [];
  }
}

function readViewed(): string[] {
  try {
    const raw = window.sessionStorage.getItem(VIEWED_STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((v): v is string => typeof v === 'string');
  } catch (error) {
    console.warn(
      `error reading sessionStorage key "${VIEWED_STORAGE_KEY}":`,
      error,
    );
    return [];
  }
}

/**
 * Given the ids already viewed this session, returns the next viewed list if
 * `id` is newly seen, or `null` when it was already counted. Keeps the
 * once-per-session dedupe rule independently testable.
 */
export function nextViewedAfter(viewed: string[], id: string): string[] | null {
  return viewed.includes(id) ? null : [...viewed, id];
}

/** Whether `now` falls within the announcement's [startDate, endDate] window. */
export function isWithinWindow(a: Announcement, now: number): boolean {
  if (a.startDate) {
    const start = Date.parse(a.startDate);
    if (!Number.isNaN(start) && now < start) return false;
  }
  if (a.endDate) {
    const end = Date.parse(a.endDate);
    if (!Number.isNaN(end) && now > end) return false;
  }
  return true;
}

/**
 * Selects the announcements that should be visible to this user — those inside
 * their date window that haven't been dismissed — and persists dismissals to
 * localStorage. `isReady` is false until we've read localStorage, so callers can
 * avoid a flash of a card the user already dismissed.
 *
 * Also exposes engagement tracking (`trackClick`, `trackView`) and emits a
 * `dismissed` event from `dismiss`, all keyed by announcement id, following the
 * shared `trackEvent`/`useTrackingUser` pattern.
 */
export function useAnnouncements(): {
  announcements: Announcement[];
  dismiss: (id: string) => void;
  trackClick: (id: string) => void;
  trackView: (id: string) => void;
  isReady: boolean;
} {
  const [dismissed, setDismissed] = useState<string[]>([]);
  const [isReady, setIsReady] = useState(false);
  const trackingUser = useTrackingUser();

  useEffect(() => {
    setDismissed(readDismissed());
    setIsReady(true);
  }, []);

  const dismiss = useCallback(
    (id: string) => {
      const isNew = !dismissed.includes(id);
      setDismissed((prev) => {
        if (prev.includes(id)) return prev;
        const next = [...prev, id];
        try {
          window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
        } catch (error) {
          console.warn(
            `error setting localStorage key "${STORAGE_KEY}":`,
            error,
          );
        }
        return next;
      });
      // Emit outside the updater (updaters must be pure / can run twice in
      // StrictMode) and only on a genuinely new dismissal, so we never
      // double-count.
      if (isNew && trackingUser) {
        trackEvent({
          name: 'app/announcement.dismissed',
          data: { id },
          user: trackingUser,
          v: TRACK_VERSION,
        });
      }
    },
    [dismissed, trackingUser],
  );

  const trackClick = useCallback(
    (id: string) => {
      if (!trackingUser) return;
      const cta = allAnnouncements.find((a) => a.id === id)?.cta;
      trackEvent({
        name: 'app/announcement.clicked',
        data: { id, cta: cta?.label, href: cta?.href },
        user: trackingUser,
        v: TRACK_VERSION,
      });
    },
    [trackingUser],
  );

  const trackView = useCallback(
    (id: string) => {
      if (!trackingUser) return;
      const next = nextViewedAfter(readViewed(), id);
      if (!next) return;
      try {
        window.sessionStorage.setItem(VIEWED_STORAGE_KEY, JSON.stringify(next));
      } catch (error) {
        console.warn(
          `error setting sessionStorage key "${VIEWED_STORAGE_KEY}":`,
          error,
        );
      }
      trackEvent({
        name: 'app/announcement.viewed',
        data: { id },
        user: trackingUser,
        v: TRACK_VERSION,
      });
    },
    [trackingUser],
  );

  const announcements = useMemo(() => {
    // Computed once per render; the window only needs to be evaluated when the
    // sidebar mounts, so a fresh `now` on render is sufficient.
    const now = Date.now();
    const dismissedSet = new Set(dismissed);
    // De-dupe by id: ids are dismissal keys (and React list keys), so a repeated
    // id would dismiss/render as a single card. Keep the first occurrence.
    const seen = new Set<string>();
    return allAnnouncements.filter((a) => {
      if (seen.has(a.id)) return false;
      seen.add(a.id);
      return !dismissedSet.has(a.id) && isWithinWindow(a, now);
    });
  }, [dismissed]);

  return useMemo(
    () => ({ announcements, dismiss, trackClick, trackView, isReady }),
    [announcements, dismiss, trackClick, trackView, isReady],
  );
}
