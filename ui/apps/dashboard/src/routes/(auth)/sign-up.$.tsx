import SplitView from '@/components/SignIn/SplitView';
import { canonicalLink } from '@/utils/urls';
import { SignUp } from '@clerk/tanstack-react-start';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { createFileRoute } from '@tanstack/react-router';

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
  head: () => ({
    links: [canonicalLink('/sign-up')],
  }),
});

function RouteComponent() {
  const anonymousId = getAnonymousId();

  return (
    <SplitView>
      <div className="my-auto flex w-full flex-col items-center">
        {/* Rendered here rather than through Clerk's `logoImageUrl` so the
            mark comes from the SVG component, which inherits `currentColor`
            and needs no dark-mode filter. */}
        <InngestLogo className="text-basis mb-8" width={132} />

        <SignUp
          unsafeMetadata={{
            ...(anonymousId && { anonymousID: anonymousId }),
          }}
          appearance={{
            elements: {
              footer: 'bg-none',
              logoBox: 'hidden',
            },
          }}
        />

        <p className="text-subtle mt-6 max-w-xs text-center text-xs">
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
      </div>
    </SplitView>
  );
}
