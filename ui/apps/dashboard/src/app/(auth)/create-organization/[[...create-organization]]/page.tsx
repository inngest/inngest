import { revalidatePath } from 'next/cache';
import { CreateOrganization } from '@clerk/nextjs';

import SplitView from '@/app/(auth)/SplitView';

export default function CreateOrganizationPage() {
  // We run revalidatePath to clear Next.js cache so that the user doesn't get stale data from a
  // previous organization if they switch organizations.
  revalidatePath('/', 'layout');

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <CreateOrganization afterCreateOrganizationUrl="/create-organization/set-up" />
      </div>
    </SplitView>
  );
}
