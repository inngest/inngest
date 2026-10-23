import { useEffect, useRef } from 'react';
import { useUser } from '@clerk/tanstack-react-start';

import LoadingIcon from '@/components/Icons/LoadingIcon';

type ClerkUser = NonNullable<ReturnType<typeof useUser>['user']>;

type ReloadClerkAndRedirectProps = {
  redirectURL: string;
  beforeRedirect?: (user: ClerkUser) => Promise<void>;
};

// Refresh server-side Clerk changes before navigating.
export default function ReloadClerkAndRedirect({
  redirectURL,
  beforeRedirect,
}: ReloadClerkAndRedirectProps) {
  const { isLoaded, user } = useUser();
  // Callback changes should not restart the redirect.
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
