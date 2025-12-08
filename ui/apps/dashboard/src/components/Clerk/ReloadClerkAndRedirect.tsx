import { useEffect } from 'react';
import { useUser } from '@clerk/tanstack-react-start';

import LoadingIcon from '@/components/Icons/LoadingIcon';

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
export default function ReloadClerkAndRedirect({
  redirectURL,
}: ReloadClerkAndRedirectProps) {
  const { isLoaded, user } = useUser();

  useEffect(() => {
    if (!isLoaded) return;

    user?.reload().then(() => {
      window.location.replace(redirectURL);
    });
  }, [isLoaded, redirectURL]);

  return (
    <div className="flex h-full w-full items-center justify-center">
      <LoadingIcon />
    </div>
  );
}
