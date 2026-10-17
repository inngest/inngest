import { useEffect, useRef } from 'react';
import { useUser } from '@clerk/tanstack-react-start';

import LoadingIcon from '@/components/Icons/LoadingIcon';

type ClerkUser = NonNullable<ReturnType<typeof useUser>['user']>;

type ReloadClerkAndRedirectProps = {
  redirectURL: string;
  /**
   * Runs after Clerk has reloaded the user and before the redirect, which
   * waits for it. Errors are swallowed so they can never block the redirect.
   */
  beforeRedirect?: (user: ClerkUser) => Promise<void>;
};

/**
 * This is used to reload Clerk on the client before redirecting to a new page. This is needed when
 * we update some Clerk data on the server and need to ensure that the client has the latest data
 * before redirecting.
 *
 * @param {string} redirectURL - The URL to redirect to after reloading Clerk
 */
export default function ReloadClerkAndRedirect({
  redirectURL,
  beforeRedirect,
}: ReloadClerkAndRedirectProps) {
  const { isLoaded, user } = useUser();
  // Held in a ref so a new callback identity on re-render doesn't re-run the
  // reload/redirect effect.
  const beforeRedirectRef = useRef(beforeRedirect);
  beforeRedirectRef.current = beforeRedirect;

  useEffect(() => {
    if (!isLoaded) return;

    user?.reload().then(async (reloadedUser) => {
      try {
        await beforeRedirectRef.current?.(reloadedUser);
      } catch {
        // Never block the redirect.
      }
      window.location.replace(redirectURL);
    });
  }, [isLoaded, redirectURL]);

  return (
    <div className="flex h-full w-full items-center justify-center">
      <LoadingIcon />
    </div>
  );
}
