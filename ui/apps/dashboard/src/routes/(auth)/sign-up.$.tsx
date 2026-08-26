import LogoWall from '@/components/SignIn/LogoWall';
import SplitView from '@/components/SignIn/SplitView';
import TrustPanel from '@/components/SignIn/TrustPanel';
import { canonicalLink } from '@/utils/urls';
import { SignUp } from '@clerk/tanstack-react-start';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { Link, createFileRoute, useLocation } from '@tanstack/react-router';

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
  const { pathname } = useLocation();

  // Clerk renders its "Already have an account?" action inside the card, which
  // puts it above our trust and legal copy. On the first step we hide that
  // footer and re-render the link below them instead, so the reading order is
  // CTA -> certification -> terms -> sign in. Later steps of the flow (email
  // verification and friends) live on deeper paths and keep Clerk's own
  // footer, which carries their back and alternate-method actions.
  const isStartStep = pathname === '/sign-up' || pathname === '/sign-up/';

  return (
    <SplitView panel={<TrustPanel />}>
      <div className="my-auto flex w-full flex-col items-center gap-10">
        <div className="flex w-full flex-col items-center">
          {/* Rendered here rather than through Clerk's `logoImageUrl` so the
              mark comes from the SVG component, which inherits `currentColor`
              and needs no dark-mode filter. */}
          <InngestLogo className="text-basis mb-6" width={132} />

          <SignUp
            unsafeMetadata={{
              ...(anonymousId && { anonymousID: anonymousId }),
            }}
            appearance={{
              elements: {
                footer: isStartStep ? 'hidden' : 'bg-none',
                logoBox: 'hidden',
                // `!` because the provider sets `my-9` on the same descriptor;
                // Tailwind utilities share specificity, so source order rather
                // than class order would otherwise decide the winner.
                header: '!mb-6 !mt-0',
              },
            }}
          />

          {/* Sentence case + `uppercase` rather than literal capitals, so
              screen readers do not spell the line out letter by letter. */}
          <p className="text-muted mt-6 text-center font-mono text-[11px] uppercase tracking-wider">
            SOC 2 Type II certified &middot; Free tier, no card required
          </p>

          <p className="text-subtle mt-4 max-w-xs text-center text-xs">
            By signing up, you agree to our{' '}
            <a
              className="text-link hover:underline"
              href="https://inngest.com/terms"
              target="_blank"
              rel="noopener noreferrer"
            >
              Terms of Service
            </a>{' '}
            and{' '}
            <a
              className="text-link hover:underline"
              href="https://inngest.com/privacy"
              target="_blank"
              rel="noopener noreferrer"
            >
              Privacy Policy
            </a>
            .
          </p>

          {isStartStep && (
            <p className="text-basis mt-8 text-center text-sm">
              Already have an account?{' '}
              <Link
                to="/sign-in/$"
                params={{ _splat: '' }}
                className="text-link hover:underline"
              >
                Sign in
              </Link>
            </p>
          )}
        </div>

        {/* The trust panel is hidden below `sm`, so the logo wall is repeated
            here to keep social proof on small screens. */}
        <LogoWall className="px-4 sm:hidden" />
      </div>
    </SplitView>
  );
}
