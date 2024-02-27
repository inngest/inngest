import { OrganizationList } from '@clerk/nextjs';

import SplitView from '@/app/(auth)/SplitView';

type OrganizationListPageProps = {
  searchParams: { [key: string]: string | string[] | undefined };
};

export default async function OrganizationListPage({ searchParams }: OrganizationListPageProps) {
  const redirectURL =
    typeof searchParams.redirect_url === 'string'
      ? searchParams.redirect_url
      : process.env.NEXT_PUBLIC_HOME_PATH;

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <OrganizationList
          hidePersonal={true}
          afterCreateOrganizationUrl="/create-organization/set-up"
          afterSelectOrganizationUrl={redirectURL}
        />
      </div>
    </SplitView>
  );
}
