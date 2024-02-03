import Link from 'next/link';
import { SignIn } from '@clerk/nextjs';

import SplitView from '@/app/(logged-out)/SplitView';
import { Alert } from '@/components/Alert';
import signInRedirectErrors from './SignInRedirectErrors';

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
