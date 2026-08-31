import LoadingIcon from '@/components/Icons/LoadingIcon';
import SignInRedirectErrors, {
  hasErrorMessage,
} from '@/components/SignIn/Errors';
import SplitView from '@/components/SignIn/SplitView';
import { validateRedirectUrlSearch } from '@/lib/deepLinkUtils';
import { canonicalLink } from '@/utils/urls';
import { SignIn } from '@clerk/tanstack-react-start';
import { Alert } from '@inngest/components/Alert';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { createFileRoute, useLocation } from '@tanstack/react-router';

type SignInSearchParams = ReturnType<typeof validateRedirectUrlSearch> & {
  error?: string;
};

export const Route = createFileRoute('/(auth)/sign-in/$')({
  component: RouteComponent,
  head: () => ({
    links: [canonicalLink('/sign-in')],
  }),
  validateSearch: (search: Record<string, unknown>): SignInSearchParams => {
    return {
      ...validateRedirectUrlSearch(search),
      error: typeof search?.error === 'string' ? search.error : undefined,
    };
  },
});

function RouteComponent() {
  const { error } = Route.useSearch();
  const location = useLocation();
  const isRedirect = !location.pathname.startsWith('/sign-in');

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        {isRedirect ? (
          <div className="flex items-center justify-center">
            <LoadingIcon />
          </div>
        ) : (
          <div className="flex flex-col items-center">
            {/* Matches sign-up: the mark is rendered here rather than through
                Clerk's `logoImageUrl`, so it inherits `currentColor` instead
                of relying on a dark-mode filter over a PNG. */}
            <InngestLogo className="text-basis mb-8" width={132} />
            <SignIn
              appearance={{
                elements: {
                  footer: 'bg-none',
                  logoBox: 'hidden',
                },
              }}
            />
          </div>
        )}
        {typeof error === 'string' && (
          <Alert severity="error" className="mx-auto max-w-xs">
            <p className="text-balance">
              {hasErrorMessage(error) ? SignInRedirectErrors[error] : error}
            </p>
            <p className="mt-2">
              <Alert.Link
                size="medium"
                severity="error"
                href="https://support.inngest.com"
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
