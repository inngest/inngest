'use client';

import { usePathname } from 'next/navigation';
import { UserProfile } from '@clerk/nextjs';
import { NewButton } from '@inngest/components/Button';

export default function UserSettingsPage() {
  const pathname = usePathname();
  const security = pathname.includes('security');

  return (
    <div className="flex flex-col justify-start">
      <UserProfile
        routing="path"
        path="/settings/user"
        appearance={{
          layout: {
            logoPlacement: 'none',
          },
          elements: {
            navbar: 'hidden',
            scrollBox: 'bg-white shadow-none',
            pageScrollBox: 'pt-6 px-2',
          },
        }}
      />
      <NewButton
        kind="secondary"
        appearance="outlined"
        href={security ? '/settings/user' : '/settings/user/security'}
        label={security ? 'User Profile' : 'Account Security'}
        className="mb-2 mt-1"
      />
    </div>
  );
}
