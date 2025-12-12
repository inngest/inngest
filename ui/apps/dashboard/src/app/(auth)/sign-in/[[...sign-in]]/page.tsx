'use client';

import { useEffect } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { SignIn, useAuth } from '@clerk/nextjs';
import { Alert } from '@inngest/components/Alert';

import SplitView from '@/app/(auth)/SplitView';
import LoadingIcon from '@/icons/LoadingIcon';
import signInRedirectErrors from './SignInRedirectErrors';

//
// with poor man's arbitrary redirect protection
const resolveRedirect = (redirectUrl: string | null) => {
  const redirect = process.env.NEXT_PUBLIC_HOME_PATH ?? '/';

  if (typeof window === 'undefined' || !redirectUrl) {
    return redirect;
  }

  try {
    const url = new URL(redirectUrl, window.location.origin);
    return url.origin === window.location.origin
      ? `${url.pathname}${url.search}${url.hash}`
      : redirect;
  } catch {
    return redirect;
  }
};

const signInRedirectErrorMessages = {
  [signInRedirectErrors.Unauthenticated]: 'Could not resume your session. Please sign in again.',
} as const;

export default function SignInPage() {
  const { isLoaded, isSignedIn } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const redirectTo = resolveRedirect(searchParams.get('redirect_url'));

  const error = searchParams.get('error');
  const nestedRoute = pathname !== '/sign-in';

  useEffect(() => {
    //
    // Clerk redirects are not reliable. Do it ourselves.
    if (!isLoaded || !isSignedIn || nestedRoute) {
      return;
    }

    router.replace(redirectTo);
  }, [isLoaded, isSignedIn, router, redirectTo, nestedRoute]);

  function hasErrorMessage(error: string): error is keyof typeof signInRedirectErrorMessages {
    return error in signInRedirectErrorMessages;
  }

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        {isLoaded && isSignedIn && !nestedRoute ? (
          <div className="flex items-center justify-center">
            <LoadingIcon />
          </div>
        ) : (
          <SignIn
            appearance={{
              elements: {
                footer: 'bg-none',
              },
            }}
          />
        )}
        {typeof error === 'string' && (
          <Alert severity="error" className="mx-auto max-w-xs">
            <p className="text-balance">
              {hasErrorMessage(error) ? signInRedirectErrorMessages[error] : error}
            </p>
            <p className="mt-2">
              <Alert.Link
                size="medium"
                severity="error"
                href="/support"
                className="inline underline"
              >
                Contact support
              </Alert.Link>{' '}
              if this problem persists.
            </p>
          </Alert>
        )}
      </div>
    </SplitView>
  );
}
