'use client';

import { useEffect, useState } from 'react';
import { usePathname, useSearchParams, useSelectedLayoutSegments } from 'next/navigation';
import Script from 'next/script';
import { useOrganization, useUser } from '@clerk/nextjs';

declare global {
  interface Window {
    inngest: {
      init: (key: string, options: Record<string, any>) => void;
      send: (event: Record<string, any>) => void;
    };
  }
}

export default function PageViewTracker() {
  const { user, isSignedIn } = useUser();
  const { organization } = useOrganization();
  const [lastUrl, setLastUrl] = useState<string>();
  const [isInitialized, setInitialized] = useState(false);
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const segments = useSelectedLayoutSegments();

  useEffect(
    function () {
      if (!isSignedIn || !isInitialized) {
        return;
      }
      // This effect may fire multiple times for the same URL, so we compare vs. last tracked URL
      const url = `${pathname}?${searchParams.toString()}`;
      if (url === lastUrl) {
        return;
      }
      setLastUrl(url);
      const ref = searchParams.get('ref') || null;
      // NOTE - This may fire before the entire view loads
      window.inngest.send({
        name: 'app/page.viewed',
        data: {
          routeSegments: segments,
          ref,
        },
        user: {
          external_id: user.externalId,
          email: user.primaryEmailAddress?.emailAddress,
          name: user.fullName,
          ...(!!organization?.publicMetadata.accountID && {
            account_id: organization.publicMetadata.accountID,
          }),
          screen_resolution: `${window.screen.width * window.devicePixelRatio}x${
            window.screen.height * window.devicePixelRatio
          }`,
          screen_size: `${window.screen.width}x${window.screen.height}`,
        },
        v: '2023-05-11.1',
      });
    },
    [
      lastUrl,
      segments,
      pathname,
      searchParams,
      isInitialized,
      isSignedIn,
      user?.externalId,
      user?.primaryEmailAddress?.emailAddress,
      user?.fullName,
      organization?.publicMetadata.accountID,
    ]
  );

  function onScriptLoad() {
    const options: { host?: string } = {};
    if (process.env.NEXT_PUBLIC_EVENT_API_HOST) {
      options.host = process.env.NEXT_PUBLIC_EVENT_API_HOST;
    }
    if (!process.env.NEXT_PUBLIC_INNGEST_EVENT_KEY) {
      if (process.env.NODE_ENV !== 'production') {
        console.warn('Set NEXT_PUBLIC_INNGEST_EVENT_KEY to track page views');
      }
      return;
    }
    window.inngest.init(process.env.NEXT_PUBLIC_INNGEST_EVENT_KEY, options);

    setInitialized(true);
  }

  return (
    <>
      <Script src="https://unpkg.com/@inngest/browser/inngest.min.js" onLoad={onScriptLoad} />
    </>
  );
}
