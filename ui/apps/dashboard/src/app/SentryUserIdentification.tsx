'use client';

import { useEffect } from 'react';
import { useUser } from '@clerk/nextjs';
import * as Sentry from '@sentry/nextjs';

export default function SentryUserIdentification() {
  const { user } = useUser();

  useEffect(() => {
    if (!user?.externalId) return;

    Sentry.setUser({
      id: user.externalId,
      clerk_user_id: user.id,
      email: user.primaryEmailAddress?.emailAddress,
    });
  }, [user?.externalId, user?.id, user?.primaryEmailAddress?.emailAddress]);

  return null;
}
