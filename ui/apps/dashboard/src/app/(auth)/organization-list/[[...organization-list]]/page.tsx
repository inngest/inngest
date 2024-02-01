import { OrganizationList } from '@clerk/nextjs';

import SplitView from '@/app/(logged-out)/SplitView';

export default function OrganizationListPage() {
  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <OrganizationList
          hidePersonal={true}
          afterSelectOrganizationUrl={process.env.NEXT_PUBLIC_HOME_PATH}
          appearance={{
            elements: {
              // This hides the "Create Organization" button until this feature is ready.
              dividerRow: 'hidden',
              button: 'hidden',
            },
          }}
        />
      </div>
    </SplitView>
  );
}
