import { ProfileMenu } from './ProfileMenu';

export type ProfileType = { orgName?: string; fullName: string };

export const Profile = ({ collapsed, profile }: { collapsed: boolean; profile: ProfileType }) => {
  return (
    <ProfileMenu>
      <div
        className={`border-subtle flex h-16 w-full flex-row items-center justify-start border-t px-2.5 ${
          collapsed ? 'justify-center' : 'justify-start'
        }`}
      >
        <div
          className={`hover:bg-canvasSubtle flex w-full flex-row items-center rounded p-1 ${
            collapsed ? 'justify-center' : 'justify-start'
          }`}
        >
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
