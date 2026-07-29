import { useEffect, useState } from 'react';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';

import AnnouncementCard from './AnnouncementCard';
import { useAnnouncements } from './useAnnouncements';

// How many cards are rendered at once. The rest exist but stay off-stack until
// one ahead of them is dismissed.
const MAX_VISIBLE = 3;

// Per-depth resting transform. depth 0 is the front card; deeper cards sit
// smaller, higher (peeking above the front's top edge), faded and slightly
// fanned, so the stack reads as a pile of cards without needing a counter.
// y offsets must exceed the height lost to scaling (origin is bottom-center) so
// each card's top edge actually peeks above the one in front of it.
const DEPTH_STYLES = [
  { scale: 1, y: 0, opacity: 1, rotate: 0 },
  { scale: 0.9, y: -30, opacity: 0.9, rotate: 0 },
  { scale: 0.7, y: -15, opacity: 0.9, rotate: 0 },
] as const;

// A soft spring gives the flip/dismiss a settled "shuffle" feel rather than a
// quick linear slide.
const SHUFFLE_SPRING = {
  type: 'spring' as const,
  stiffness: 200,
  damping: 20,
  mass: 0.9,
};

export default function AnnouncementStack({
  collapsed,
}: {
  collapsed: boolean;
}) {
  const { announcements, dismiss, trackClick, trackView, isReady } =
    useAnnouncements();
  const reduceMotion = useReducedMotion();
  const [activeIndex, setActiveIndex] = useState(0);

  // Keep the front pointer in range as the list shrinks. Dismissing the front
  // card lets the next one slide into the same index automatically; this only
  // fires when we dismissed the last card and need to wrap to the start.
  useEffect(() => {
    if (announcements.length > 0 && activeIndex >= announcements.length) {
      setActiveIndex(0);
    }
  }, [announcements.length, activeIndex]);

  // Count each card that surfaces to the front as a view (deduped per session
  // inside trackView). Skip while collapsed or pre-hydration — nothing is shown.
  const frontId =
    announcements.length > 0
      ? announcements[activeIndex % announcements.length]?.id
      : undefined;
  useEffect(() => {
    if (collapsed || !isReady || !frontId) return;
    trackView(frontId);
  }, [collapsed, isReady, frontId, trackView]);

  // Hidden when collapsed, before hydration, or once everything is dismissed.
  if (collapsed || !isReady || announcements.length === 0) {
    return null;
  }

  const n = announcements.length;
  const safeIndex = activeIndex % n;
  const transition = reduceMotion ? { duration: 0 } : SHUFFLE_SPRING;

  const stack = announcements
    .map((announcement, i) => ({
      announcement,
      index: i,
      depth: (i - safeIndex + n) % n,
    }))
    .filter(({ depth }) => depth < MAX_VISIBLE);

  return (
    // Fixed height = one card (h-[192px] content + border ≈ 194px; padding is
    // now inside the content) plus headroom for the peeking cards fanned above
    // it. Keep in sync with CARD_CONTENT_HEIGHT_CLASS in AnnouncementCard.
    <div className="relative mb-3 h-[228px]">
      <AnimatePresence initial={false}>
        {stack.map(({ announcement, index, depth }) => {
          const target = DEPTH_STYLES[Math.min(depth, DEPTH_STYLES.length - 1)];
          const isFront = depth === 0;
          return (
            <motion.div
              key={announcement.id}
              className={`absolute inset-x-0 bottom-0 ${
                isFront ? '' : 'cursor-pointer'
              }`}
              style={{ transformOrigin: 'bottom center', zIndex: 30 - depth }}
              initial={{ opacity: 0, scale: 0.85, y: -28, rotate: 0 }}
              animate={{
                scale: target.scale,
                y: target.y,
                opacity: target.opacity,
                rotate: target.rotate,
              }}
              exit={{
                opacity: 0,
                scale: 0.92,
                y: 16,
                transition: {
                  duration: reduceMotion ? 0 : 0.28,
                  ease: [0.4, 0, 1, 1],
                },
              }}
              transition={transition}
              aria-hidden={!isFront}
              onClick={isFront ? undefined : () => setActiveIndex(index)}
            >
              <AnnouncementCard
                announcement={announcement}
                isFront={isFront}
                onDismiss={isFront ? () => dismiss(announcement.id) : undefined}
                onCtaClick={
                  isFront ? () => trackClick(announcement.id) : undefined
                }
              />
            </motion.div>
          );
        })}
      </AnimatePresence>
    </div>
  );
}
