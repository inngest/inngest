import { useCallback, useEffect, useMemo, useState } from 'react';

import {
  announcements as allAnnouncements,
  type Announcement,
} from './announcements';

const STORAGE_KEY = 'dismissedAnnouncements';

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
 */
export function useAnnouncements(): {
  announcements: Announcement[];
  dismiss: (id: string) => void;
  isReady: boolean;
} {
  const [dismissed, setDismissed] = useState<string[]>([]);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    setDismissed(readDismissed());
    setIsReady(true);
  }, []);

  const dismiss = useCallback((id: string) => {
    setDismissed((prev) => {
      if (prev.includes(id)) return prev;
      const next = [...prev, id];
      try {
        window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
      } catch (error) {
        console.warn(`error setting localStorage key "${STORAGE_KEY}":`, error);
      }
      return next;
    });
  }, []);

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
    () => ({ announcements, dismiss, isReady }),
    [announcements, dismiss, isReady],
  );
}
