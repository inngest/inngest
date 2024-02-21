'use client';

import { useEffect } from 'react';
import { useUser } from '@clerk/nextjs';

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
  const { user } = useUser();

  useEffect(() => {
    user?.reload().then(() => {
      window.location.href = redirectURL;
    });
  }, [user, redirectURL]);

  return null;
}
