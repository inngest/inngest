'use client';

import { UserProfile } from '@clerk/nextjs';

export default function UserSettingsPage() {
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
      >
        <UserProfile.Page label="security" />
      </UserProfile>
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
            profileSectionItemList__activeDevices: 'h-24 overflow-y-scroll',
          },
        }}
      >
        <UserProfile.Page label="account" />
      </UserProfile>
    </div>
  );
}
