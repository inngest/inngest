import { type Metadata } from 'next';
import Link from 'next/link';
import { SignIn } from '@clerk/nextjs';
import { Alert } from '@inngest/components/Alert';

import SplitView from '@/app/(auth)/SplitView';
import signInRedirectErrors from './SignInRedirectErrors';

export const metadata: Metadata = {
  title: 'Sign in - Inngest Cloud',
  description: 'Sign into your account',
  alternates: {
    canonical: new URL(
      '/sign-in',
      process.env.NEXT_PUBLIC_APP_URL || 'https://app.inngest.com'
    ).toString(),
  },
};

const signInRedirectErrorMessages = {
  [signInRedirectErrors.Unauthenticated]: 'Could not resume your session. Please sign in again.',
} as const;

type SignInPageProps = {
  searchParams: { [key: string]: string | string[] | undefined };
};

export default function SignInPage({ searchParams }: SignInPageProps) {
  function hasErrorMessage(error: string): error is keyof typeof signInRedirectErrorMessages {
    return error in signInRedirectErrorMessages;
  }

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <SignIn />
        {typeof searchParams.error === 'string' && (
          <Alert severity="error" className="mx-auto max-w-xs">
            <p className="text-balance">
              {hasErrorMessage(searchParams.error)
                ? signInRedirectErrorMessages[searchParams.error]
                : searchParams.error}
            </p>
            <p className="mt-2">
              <Link href="/support" className="underline">
                Contact support
              </Link>{' '}
              if this problem persists.
            </p>
          </Alert>
        )}
      </div>
    </SplitView>
  );
}
