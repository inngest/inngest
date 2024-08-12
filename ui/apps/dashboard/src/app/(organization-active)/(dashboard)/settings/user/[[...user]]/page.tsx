import { UserProfile } from '@clerk/nextjs';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';

export default async function UserSettingsPage() {
  const newIANav = await getBooleanFlag('new-ia-nav');

  return (
    <UserProfile
      routing="path"
      path="/settings/user"
      appearance={{
        elements: {
          rootBox: 'h-full',
          card: newIANav ? '' : 'divide-x divide-slate-100',
          navbar: newIANav ? 'hidden' : 'p-8 border-none',
          scrollBox: 'bg-white',
          pageScrollBox: '[scrollbar-width:none]', // hides the Clerk's scrollbar
        },
      }}
    />
  );
}
