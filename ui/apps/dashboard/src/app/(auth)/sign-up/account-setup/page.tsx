'use client';

import { useEffect, useState } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { useUser } from '@clerk/nextjs';
import { ExclamationCircleIcon } from '@heroicons/react/20/solid';

import LoadingIcon from '@/icons/LoadingIcon';

export default function AccountSetupPage() {
  const router = useRouter();
  const secondsElapsed = useSecondsElapsed();
  const { user } = useUser();

  const isAccountSetup = user && user.externalId && user.publicMetadata.accountID;

  // Reload the user until the account is set up
  useEffect(() => {
    if (!user || isAccountSetup) return;

    const intervalID = setInterval(() => {
      user.reload();
    }, 500);

    return () => {
      clearInterval(intervalID);
    };
  }, [isAccountSetup, user]);

  // Redirect to the home page once the account is set up
  useEffect(() => {
    if (!isAccountSetup) return;

    // We use `replace` so that the user doesn't get redirected back to this page if they click the back button
    router.replace(process.env.NEXT_PUBLIC_HOME_PATH as Route);
  }, [isAccountSetup, router]);

  useEffect(() => {
    if (secondsElapsed !== 10) return;

    window.inngest.send({
      name: 'app/account.setup.delayed',
      data: {
        userID: user?.id,
        delayInSeconds: secondsElapsed,
      },
      user,
      v: '2023-08-31.1',
    });
  }, [secondsElapsed, user]);

  if (secondsElapsed > 30) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-5">
        <div className="inline-flex items-center gap-2 text-red-600">
          <ExclamationCircleIcon className="h-4 w-4" />
          <h2 className="text-sm">Failed to set up your account</h2>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full w-full items-center justify-center">
      <LoadingIcon />
    </div>
  );
}

function useSecondsElapsed() {
  const [seconds, setSeconds] = useState(0);
  useEffect(() => {
    const intervalID = setInterval(() => {
      setSeconds((seconds) => seconds + 1);
    }, 1_000);

    return () => {
      clearInterval(intervalID);
    };
  }, []);

  return seconds;
}
