import SplitView from '@/components/SignIn/SplitView';
import { SignUp } from '@clerk/tanstack-react-start';
import { createFileRoute } from '@tanstack/react-router';
import logoImageUrl from '@inngest/components/icons/logos/inngest-logo-black.png';

const getAnonymousId = () => {
  if (typeof document === 'undefined') {
    return null;
  }

  const cookie = document.cookie
    .split('; ')
    .find((c) => c.startsWith('ajs_anonymous_id='));

  return cookie ? cookie.split('=')[1] : null;
};

export const Route = createFileRoute('/(auth)/sign-up/$')({
  component: RouteComponent,
});

function RouteComponent() {
  const anonymousId = getAnonymousId();

  return (
    <SplitView>
      <div className="mx-auto my-8 mt-auto text-center">
        <SignUp
          unsafeMetadata={{
            ...(anonymousId && { anonymousID: anonymousId }),
          }}
          appearance={{
            layout: {
              logoImageUrl,
            },
            elements: {
              footer: 'bg-none',
              logoBox: 'flex m-0 justify-center',
              logoImage: 'max-h-16 w-auto object-contain dark:invert',
            },
          }}
        />
      </div>
      <p className="text-subtle mt-auto text-center text-xs">
        By signing up, you agree to our{' '}
        <a
          className="text-link hover:underline"
          href="https://inngest.com/terms"
          target="_blank"
        >
          Terms of Service
        </a>{' '}
        and{' '}
        <a
          className="text-link hover:underline"
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
