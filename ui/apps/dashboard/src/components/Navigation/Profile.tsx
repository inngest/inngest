import { ProfileMenu } from './ProfileMenu';

export type ProfileType = { orgName?: string; fullName: string };

export const Profile = ({ collapsed, profile }: { collapsed: boolean; profile: ProfileType }) => {
  return (
    <ProfileMenu>
      <div className="border-subtle flex h-16 flex-row items-center justify-start border-t px-4">
        <div className="hover:bg-canvasSubtle flex w-full flex-row items-center justify-start rounded p-1 px-1.5">
          <div className="bg-canvasMuted text-muted flex h-8 w-8 items-center justify-center rounded-full text-xs uppercase">
            {profile.orgName?.substring(0, 2) || '?'}
          </div>
          {!collapsed && (
            <div className="ml-2 flex flex-col items-start justify-start">
              <div className="text-muted leading-1 text-sm">{profile.orgName}</div>
              <div className="text-subtle text-xs leading-4">{profile.fullName}</div>
            </div>
          )}
        </div>
      </div>
    </ProfileMenu>
  );
};
