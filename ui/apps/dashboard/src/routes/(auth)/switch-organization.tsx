import LoadingIcon from '@/components/Icons/LoadingIcon';
import SplitView from '@/components/SignIn/SplitView';
import { createFileRoute } from '@tanstack/react-router';
import { useClerk } from '@clerk/tanstack-react-start';
import { useEffect } from 'react';

type SwitchOrganizationSearchParams = {
  organization_id?: string;
  redirect_url?: string;
};

export const Route = createFileRoute('/(auth)/switch-organization')({
  component: RouteComponent,
  validateSearch: (
    search: Record<string, unknown>,
  ): SwitchOrganizationSearchParams => ({
    organization_id:
      typeof search.organization_id === 'string'
        ? search.organization_id
        : undefined,
    redirect_url:
      typeof search.redirect_url === 'string' &&
      search.redirect_url.startsWith('/')
        ? search.redirect_url
        : undefined,
  }),
});

function RouteComponent() {
  const { organization_id, redirect_url } = Route.useSearch();
  const { loaded, setActive } = useClerk();

  useEffect(() => {
    if (!loaded || !organization_id || !redirect_url) {
      return;
    }

    const switchOrganization = async () => {
      await setActive({ organization: organization_id });
      window.location.replace(redirect_url);
    };

    void switchOrganization();
  }, [loaded, organization_id, redirect_url, setActive]);

  return (
    <SplitView>
      <div className="mx-auto my-auto flex items-center gap-2 text-center">
        <LoadingIcon />
        <span>Switching organizations...</span>
      </div>
    </SplitView>
  );
}
