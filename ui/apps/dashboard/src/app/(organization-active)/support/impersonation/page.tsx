'use client';

import { useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth, useClerk, useSignIn } from '@clerk/nextjs';

import LoadingIcon from '@/icons/LoadingIcon';
import { generateActorToken } from './action';

export default function ImpersonateUsers() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const userId = searchParams.get('user_id');
  const auth = useAuth();

  const { isLoaded, signIn } = useSignIn();
  const { setActive } = useClerk();

  useEffect(() => {
    if (!auth.userId || !userId) {
      router.push('/');
      return;
    }

    if (!isLoaded) return;

    const performImpersonation = async () => {
      try {
        const res = await generateActorToken(auth.userId, userId);
        if (!res.ok) {
          console.log('Error retrieving token', res.error);
          return;
        }
        const { createdSessionId } = await signIn.create({
          strategy: 'ticket',
          ticket: res.token,
        });

        await setActive({ session: createdSessionId });
        router.push('/');
      } catch (err) {
        console.error('Impersonation failed:', JSON.stringify(err, null, 2));
        router.push('/');
      }
    };

    performImpersonation();
  }, [userId, isLoaded, signIn, setActive, router, auth.userId]);

  if (!userId) {
    return null;
  }

  if (!isLoaded) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  return (
    <div className="flex h-full w-full items-center justify-center gap-2">
      <LoadingIcon />
      <span>Signing you in...</span>
    </div>
  );
}
