import { createFileRoute } from '@tanstack/react-router';
import { OrganizationList } from '@clerk/tanstack-react-start';
import SplitView from '@/components/SignIn/SplitView';

type OrganizationListSearchParams = {
  redirect_url?: string;
};

export const Route = createFileRoute('/(auth)/organization-list/$')({
  component: RouteComponent,
  validateSearch: (
    search: Record<string, unknown>,
  ): OrganizationListSearchParams => {
    return {
      redirect_url: search?.redirect_url as string | undefined,
    };
  },
});

function RouteComponent() {
  const { redirect_url } = Route.useSearch();
  const redirectURL =
    redirect_url || import.meta.env.VITE_PUBLIC_HOME_PATH || '/';

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <OrganizationList
          hidePersonal={true}
          skipInvitationScreen={true}
          afterCreateOrganizationUrl="/organization-setup"
          afterSelectOrganizationUrl={redirectURL}
        />
      </div>
    </SplitView>
  );
}
