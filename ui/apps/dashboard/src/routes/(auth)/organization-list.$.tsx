import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { OrganizationList } from '@clerk/tanstack-react-start';
import SplitView from '@/components/SignIn/SplitView';
import { useAuth } from '@clerk/tanstack-react-start';
import { useEffect, useRef } from 'react';
import LoadingIcon from '@/components/Icons/LoadingIcon';

type OrganizationListSearchParams = {
  redirect_url?: string;
};

export const Route = createFileRoute('/(auth)/organization-list/$')({
  component: RouteComponent,
  validateSearch: (
    search: Record<string, unknown>,
  ): OrganizationListSearchParams => {
    return {
      redirect_url:
        typeof search?.redirect_url === 'string' &&
        search.redirect_url.startsWith('/')
          ? search.redirect_url
          : undefined,
    };
  },
});

function RouteComponent() {
  const { redirect_url } = Route.useSearch();
  const navigate = useNavigate();
  const { isLoaded, orgId, getToken } = useAuth();
  const initialOrgIdRef = useRef<string | null | undefined>(undefined);

  const redirectURL =
    redirect_url || import.meta.env.VITE_PUBLIC_HOME_PATH || '/';

  useEffect(() => {
    if (!isLoaded) {
      return;
    }
    if (initialOrgIdRef.current !== undefined) {
      return;
    }

    initialOrgIdRef.current = orgId ?? null;
  }, [isLoaded, orgId]);

  const shouldRedirect =
    isLoaded &&
    !!orgId &&
    initialOrgIdRef.current !== undefined &&
    orgId !== initialOrgIdRef.current;

  useEffect(() => {
    if (!shouldRedirect) {
      return;
    }

    let cancelled = false;

    (async () => {
      await getToken({ skipCache: true });
      if (cancelled) {
        return;
      }

      navigate({
        to: redirectURL,
        replace: true,
      });
    })();

    return () => {
      cancelled = true;
    };
  }, [getToken, navigate, redirectURL, shouldRedirect]);

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        {shouldRedirect ? (
          <div className="flex items-center justify-center">
            <LoadingIcon />
          </div>
        ) : (
          <OrganizationList
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
