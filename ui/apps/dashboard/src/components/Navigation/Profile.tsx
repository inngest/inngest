'use client';

import { usePathname } from 'next/navigation';

import { ProfileMenu } from './ProfileMenu';

export type ProfileType = {
  displayName: string;
  isMarketplace: boolean;
  orgName?: string;
};

export const Profile = ({ collapsed, profile }: { collapsed: boolean; profile: ProfileType }) => {
  const pathname = usePathname();
  const active =
    pathname.startsWith('/settings/organization') ||
    pathname.startsWith('/billing') ||
    pathname.startsWith('/settings/user');

  return (
    <ProfileMenu isMarketplace={profile.isMarketplace}>
      <div
        className={`border-subtle mt-2 flex h-16 w-full flex-row items-center justify-start border-t px-2.5 `}
      >
        <div
          className={`flex w-full flex-row items-center rounded p-1 ${
            collapsed ? 'justify-center' : 'justify-start'
          } ${
            active
              ? 'bg-secondary-4xSubtle text-info hover:bg-secondary-3xSubtle'
              : 'hover:bg-canvasSubtle text-subtle'
          }`}
        >
          <div className="bg-canvasMuted text-subtle flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs uppercase">
            {profile.orgName?.substring(0, 2) || '?'}
          </div>

          {!collapsed && (
            <div className="ml-2 flex flex-col items-start justify-start overflow-hidden">
              <div
                className="text-subtle leading-1 max-w-full overflow-hidden text-ellipsis text-nowrap text-sm"
                title={profile.orgName}
              >
                {profile.orgName}
              </div>
              <div className="text-muted text-xs leading-4">{profile.displayName}</div>
            </div>
          )}
        </div>
      </div>
    </ProfileMenu>
  );
};
