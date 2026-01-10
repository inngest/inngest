import LoadingIcon from '@/components/Icons/LoadingIcon';
import SignInRedirectErrors, {
  hasErrorMessage,
} from '@/components/SignIn/Errors';
import SplitView from '@/components/SignIn/SplitView';
import { SignIn } from '@clerk/tanstack-react-start';
import { Alert } from '@inngest/components/Alert';
import { createFileRoute, useLocation } from '@tanstack/react-router';
import logoImageUrl from '@inngest/components/icons/logos/inngest-logo-black.png';

type SignInSearchParams = {
  redirect_url?: string;
  error?: string;
};

export const Route = createFileRoute('/(auth)/sign-in/$')({
  component: RouteComponent,
  validateSearch: (search: Record<string, unknown>): SignInSearchParams => {
    return {
      redirect_url:
        typeof search?.redirect_url === 'string' &&
        //
        // we only support relative redirects
        search.redirect_url.startsWith('/')
          ? search.redirect_url
          : undefined,
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
          <SignIn
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
