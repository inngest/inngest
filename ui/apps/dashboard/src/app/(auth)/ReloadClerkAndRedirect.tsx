'use client';

import { useEffect } from 'react';
import { useUser } from '@clerk/nextjs';

import LoadingIcon from '@/icons/LoadingIcon';

type ReloadClerkAndRedirectProps = {
  redirectURL: string;
};

/**
 * This is used to reload Clerk on the client before redirecting to a new page. This is needed when
 * we update some Clerk data on the server and need to ensure that the client has the latest data
 * before redirecting.
 *
 * @param {string} redirectURL - The URL to redirect to after reloading Clerk
 */
export default function ReloadClerkAndRedirect({ redirectURL }: ReloadClerkAndRedirectProps) {
  const { isLoaded, user } = useUser();

  useEffect(() => {
    if (!isLoaded) return;

    user?.reload().then(() => {
      window.location.replace(redirectURL);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- We don't want to run this effect when the user changes as this would cause an infinite loop
  }, [isLoaded, redirectURL]);

  return (
    <div className="flex h-full w-full items-center justify-center">
      <LoadingIcon />
    </div>
  );
}
