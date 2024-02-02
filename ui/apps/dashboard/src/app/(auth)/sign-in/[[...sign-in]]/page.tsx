import Link from 'next/link';
import { SignIn } from '@clerk/nextjs';

import SplitView from '@/app/(logged-out)/SplitView';
import { Alert } from '@/components/Alert';

type SignInPageProps = {
  searchParams: { [key: string]: string | string[] | undefined };
};

export default function SignInPage({ searchParams }: SignInPageProps) {
  const errorMessages = {
    unauthenticated: 'Could not successfully signed you in. Please sign in and try again.',
  } as const;

  function hasErrorMessage(error: string): error is keyof typeof errorMessages {
    return error in errorMessages;
  }

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <SignIn />
        {typeof searchParams.error === 'string' && (
          <Alert severity="error" className="mx-auto max-w-xs">
            <p className="text-balance">
              {hasErrorMessage(searchParams.error)
                ? errorMessages[searchParams.error]
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
