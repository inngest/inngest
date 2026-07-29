import { useEffect } from 'react';
import { useRouterState } from '@tanstack/react-router';

/**
 * Fires a Segment `page` call on the initial load and on every client-side
 * navigation. TanStack Router navigates without a full page reload, so the
 * Segment loader snippet's one-time `page()` call would otherwise miss all
 * subsequent route changes.
 *
 * The path/url are passed explicitly rather than letting Segment infer them
 * from the DOM: this effect is triggered by TanStack's router state, which
 * updates a tick before `window.location`/`document.title` are committed, so a
 * bare `page()` would read the *previous* page's location.
 */
export function SegmentPageTracking() {
  const location = useRouterState({ select: (state) => state.location });

  useEffect(() => {
    window.analytics?.page({
      path: location.pathname,
      // location.href is relative (path+search+hash); origin is stable across
      // SPA navigation, so this yields a correct absolute URL.
      url: window.location.origin + location.href,
      search: location.searchStr,
    });
  }, [location.pathname, location.href, location.searchStr]);

  return null;
}
