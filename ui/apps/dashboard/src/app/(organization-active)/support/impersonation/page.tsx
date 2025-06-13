'use client';

import { useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useClerk, useSignIn } from '@clerk/nextjs';

export default function ImpersonateUsers() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const actorToken = searchParams.get('actor_token');

  const { isLoaded, signIn } = useSignIn();
  const { setActive } = useClerk();

  useEffect(() => {
    if (!actorToken) {
      router.push('/');
      return;
    }

    if (!isLoaded) return;

    const performImpersonation = async () => {
      try {
        const { createdSessionId } = await signIn.create({
          strategy: 'ticket',
          ticket: actorToken,
        });

        await setActive({ session: createdSessionId });
        router.push('/');
      } catch (err) {
        console.error('Impersonation failed:', JSON.stringify(err, null, 2));
        router.push('/');
      }
    };

    performImpersonation();
  }, [actorToken, isLoaded, signIn, setActive, router]);

  if (!actorToken) {
    return null;
  }

  if (!isLoaded) {
    return <div>Loading...</div>;
  }

  return <div>Signing you in...</div>;
}
