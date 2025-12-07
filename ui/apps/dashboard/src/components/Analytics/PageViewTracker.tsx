import { useOrganization, useUser } from "@clerk/tanstack-react-start";
import { useLocation, useMatches, useSearch } from "@tanstack/react-router";
import { useEffect, useState } from "react";

declare global {
  interface Window {
    inngest: {
      init: (key: string, options: Record<string, any>) => void;
      send: (event: Record<string, any>) => void;
    };
  }
}

export const useRouteSegments = () => {
  const matches = useMatches();
  const last = matches.at(-1);

  if (!last) {
    return [];
  }
  return last.id.split("/").filter(Boolean);
};

export default function PageViewTracker() {
  const { user, isSignedIn } = useUser();
  const { organization } = useOrganization();
  const [lastUrl, setLastUrl] = useState<string>();
  const [isInitialized, setInitialized] = useState(false);
  const search = useSearch({ strict: false });
  const location = useLocation();
  const routeSegments = useRouteSegments();

  useEffect(
    function () {
      if (!isSignedIn || !isInitialized) {
        return;
      }

      if (location.href === lastUrl) {
        return;
      }
      setLastUrl(location.href);

      // NOTE - This may fire before the entire view loads
      window.inngest.send({
        name: "app/page.viewed",
        data: {
          routeSegments,
          ref: "ref" in search ? search.ref : null,
        },
        user: {
          external_id: user.externalId,
          email: user.primaryEmailAddress?.emailAddress,
          name: user.fullName,
          ...(!!organization?.publicMetadata.accountID && {
            account_id: organization.publicMetadata.accountID,
          }),
          screen_resolution: `${
            window.screen.width * window.devicePixelRatio
          }x${window.screen.height * window.devicePixelRatio}`,
          screen_size: `${window.screen.width}x${window.screen.height}`,
        },
        v: "2023-05-11.1",
      });
    },
    [
      lastUrl,
      search,
      routeSegments,
      isInitialized,
      isSignedIn,
      user?.externalId,
      user?.primaryEmailAddress?.emailAddress,
      user?.fullName,
      organization?.publicMetadata.accountID,
    ],
  );

  useEffect(() => {
    if (window.inngest) {
      const options: { host?: string } = {};
      const eventApiHost = import.meta.env.VITE_EVENT_API_HOST;
      const eventKey = import.meta.env.VITE_INNGEST_EVENT_KEY;

      if (eventApiHost) {
        options.host = eventApiHost;
      }
      if (!eventKey) {
        if (import.meta.env.MODE !== "production") {
          console.warn("Set VITE_INNGEST_EVENT_KEY to track page views");
        }
        return;
      }
      window.inngest.init(eventKey, options);
      setInitialized(true);
      return;
    }

    const script = document.createElement("script");
    script.src = "https://unpkg.com/@inngest/browser/inngest.min.js";
    script.async = true;

    script.onload = () => {
      const options: { host?: string } = {};
      const eventApiHost = import.meta.env.VITE_EVENT_API_HOST;
      const eventKey = import.meta.env.VITE_INNGEST_EVENT_KEY;

      if (eventApiHost) {
        options.host = eventApiHost;
      }
      if (!eventKey) {
        if (import.meta.env.MODE !== "production") {
          console.warn("Set VITE_INNGEST_EVENT_KEY to track page views");
        }
        return;
      }
      window.inngest.init(eventKey, options);
      setInitialized(true);
    };

    document.head.appendChild(script);

    return () => {
      if (script.parentNode) {
        script.parentNode.removeChild(script);
      }
    };
  }, []);

  return null;
}
