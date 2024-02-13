import { CreateOrganization } from '@clerk/nextjs';

import SplitView from '@/app/(auth)/SplitView';

export default function CreateOrganizationPage() {
  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <CreateOrganization afterCreateOrganizationUrl={process.env.NEXT_PUBLIC_HOME_PATH} />
      </div>
    </SplitView>
  );
}
