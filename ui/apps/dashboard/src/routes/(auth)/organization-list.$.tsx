import LoadingIcon from '@/components/Icons/LoadingIcon';
import SplitView from '@/components/SignIn/SplitView';
import { validateRedirectUrlSearch } from '@/lib/deepLinkUtils';
import { OrganizationList, useAuth } from '@clerk/tanstack-react-start';
import { createFileRoute, useLocation } from '@tanstack/react-router';
import logoImageUrl from '@inngest/components/icons/logos/inngest-logo-black.png';

export const Route = createFileRoute('/(auth)/organization-list/$')({
  component: RouteComponent,
  validateSearch: validateRedirectUrlSearch,
});

function RouteComponent() {
  const { redirect_url } = Route.useSearch();
  const location = useLocation();
  const { isLoaded, orgId } = useAuth();
  const isRedirect = !location.pathname.startsWith('/organization-list');
  const redirectURL =
    redirect_url || import.meta.env.VITE_PUBLIC_HOME_PATH || '/';

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        {isLoaded && orgId && isRedirect ? (
          <div className="flex items-center justify-center">
            <LoadingIcon />
          </div>
        ) : (
          <OrganizationList
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
            hidePersonal={true}
            skipInvitationScreen={true}
            afterCreateOrganizationUrl="/organization-setup"
            afterSelectOrganizationUrl={redirectURL}
          />
        )}
      </div>
    </SplitView>
  );
}
