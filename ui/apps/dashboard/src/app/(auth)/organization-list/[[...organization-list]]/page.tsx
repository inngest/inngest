import { OrganizationList } from '@clerk/nextjs';

import SplitView from '@/app/(logged-out)/SplitView';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';

type OrganizationListPageProps = {
  searchParams: { [key: string]: string | string[] | undefined };
};

export default async function OrganizationListPage({ searchParams }: OrganizationListPageProps) {
  const redirectURL =
    typeof searchParams.redirect_url === 'string'
      ? searchParams.redirect_url
      : process.env.NEXT_PUBLIC_HOME_PATH;

  const isOrganizationsEnabled = await getBooleanFlag('organizations');

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <OrganizationList
          hidePersonal={true}
          afterCreateOrganizationUrl="/sign-up/account-setup"
          afterSelectOrganizationUrl={redirectURL}
          appearance={
            isOrganizationsEnabled
              ? undefined
              : {
                  elements: {
                    // This hides the "Create Organization" button until this feature is ready.
                    dividerRow: 'hidden',
                    button: 'hidden',
                  },
                }
          }
        />
      </div>
    </SplitView>
  );
}
