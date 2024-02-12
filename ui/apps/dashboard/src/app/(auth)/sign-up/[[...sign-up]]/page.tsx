import { cookies } from 'next/headers';
import { SignUp } from '@clerk/nextjs';

import SplitView from '@/app/(auth)/SplitView';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';

export default async function SignUpPage() {
  const cookieStore = cookies();
  const anonymousIDCookie = cookieStore.get('inngest_anonymous_id');

  const isOrganizationsEnabled = await getBooleanFlag('organizations');

  return (
    <SplitView>
      <div className="mx-auto my-8 mt-auto text-center">
        <SignUp
          afterSignUpUrl={isOrganizationsEnabled ? '/organization-list' : '/sign-up/account-setup'}
          unsafeMetadata={{
            ...(anonymousIDCookie?.value && { anonymousID: anonymousIDCookie.value }),
          }}
          appearance={{
            elements: {
              // We need to hide the name fields because we don't want to overwhelm users with too
              // many fields, but we still want to allow them later to set their name if they want to.
              formFieldRow__name: 'hidden',
            },
          }}
        />
      </div>
      <p className="mt-auto text-center text-xs text-slate-400">
        By signing up, you agree to our{' '}
        <a
          className="text-indigo-400 hover:underline"
          href="https://inngest.com/terms"
          target="_blank"
        >
          Terms of Service
        </a>{' '}
        and{' '}
        <a
          className="text-indigo-400 hover:underline"
          href="https://inngest.com/privacy"
          target="_blank"
        >
          Privacy Policy
        </a>
        .
      </p>
    </SplitView>
  );
}
