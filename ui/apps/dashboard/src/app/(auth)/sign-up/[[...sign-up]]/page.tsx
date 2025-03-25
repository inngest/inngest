import { type Metadata } from 'next';
import { cookies } from 'next/headers';
import { SignUp } from '@clerk/nextjs';

import SplitView from '@/app/(auth)/SplitView';

export const metadata: Metadata = {
  title: 'Sign up - Inngest Cloud',
  description: 'Create a new account',
  alternates: {
    canonical: new URL(
      '/sign-up',
      process.env.NEXT_PUBLIC_APP_URL || 'https://app.inngest.com'
    ).toString(),
  },
};

export default async function SignUpPage() {
  const cookieStore = cookies();
  const anonymousIDCookie = cookieStore.get('inngest_anonymous_id');

  return (
    <SplitView>
      <div className="mx-auto my-8 mt-auto text-center">
        <SignUp
          unsafeMetadata={{
            ...(anonymousIDCookie?.value && { anonymousID: anonymousIDCookie.value }),
          }}
          appearance={{
            elements: {
              footer: 'bg-none',
            },
          }}
        />
      </div>
      <p className="text-subtle mt-auto text-center text-xs">
        By signing up, you agree to our{' '}
        <a className="text-link hover:underline" href="https://inngest.com/terms" target="_blank">
          Terms of Service
        </a>{' '}
        and{' '}
        <a className="text-link hover:underline" href="https://inngest.com/privacy" target="_blank">
          Privacy Policy
        </a>
        .
      </p>
    </SplitView>
  );
}
