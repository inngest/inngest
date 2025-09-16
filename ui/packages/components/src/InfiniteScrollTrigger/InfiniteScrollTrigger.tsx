'use client';

import { useEffect, useRef } from 'react';

interface InfiniteScrollTriggerProps {
  onIntersect: () => void;
  hasMore: boolean;
  isLoading: boolean;
  rootMargin?: string;
}

export function InfiniteScrollTrigger({
  onIntersect,
  hasMore,
  isLoading,
  rootMargin = '200px',
}: InfiniteScrollTriggerProps) {
  const triggerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const trigger = triggerRef.current;
    if (!trigger || !hasMore || isLoading) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const [entry] = entries;
        if (entry?.isIntersecting) {
          onIntersect();
        }
      },
      {
        rootMargin,
        threshold: 1.0,
      }
    );

    observer.observe(trigger);

    return () => {
      observer.disconnect();
    };
  }, [onIntersect, hasMore, isLoading, rootMargin]);

  if (!hasMore) return null;

  return <div ref={triggerRef} className="h-0 w-0" aria-hidden="true" />;
}
