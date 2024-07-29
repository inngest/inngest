'use client';

import { usePathname } from 'next/navigation';

import { ProfileMenu } from './ProfileMenu';

export type ProfileType = { orgName?: string; displayName: string };

export const Profile = ({ collapsed, profile }: { collapsed: boolean; profile: ProfileType }) => {
  const pathname = usePathname();
  const active =
    pathname.startsWith('/settings/organization') ||
    pathname.startsWith('/settings/billing') ||
    pathname.startsWith('/settings/user');

  return (
    <ProfileMenu>
      <div
        className={`border-subtle mt-2 flex h-16 w-full flex-row items-center justify-start border-t px-2.5 ${
          collapsed ? 'justify-center' : 'justify-start'
        }`}
      >
        <div
          className={`flex w-full flex-row items-center rounded p-1 ${
            collapsed ? 'justify-center' : 'justify-start'
          } ${
            active
              ? 'bg-secondary-4xSubtle text-info hover:bg-secondary-3xSubtle'
              : 'hover:bg-canvasSubtle text-muted'
          }`}
        >
          <div className="bg-canvasMuted text-muted flex h-8 w-8 items-center justify-center rounded-full text-xs uppercase">
            {profile.orgName?.substring(0, 2) || '?'}
          </div>
          {!collapsed && (
            <div className="ml-2 flex flex-col items-start justify-start">
              <div className="text-muted leading-1 text-sm">{profile.orgName}</div>
              <div className="text-subtle text-xs leading-4">{profile.displayName}</div>
            </div>
          )}
        </div>
      </div>
    </ProfileMenu>
  );
};
