import { RiCodeSSlashLine } from '@remixicon/react';

import { ProfileMenu } from './ProfileMenu';

const Profile = ({ collapsed }: { collapsed: boolean }) => {
  return (
    <ProfileMenu>
      <div
        className={`border-subtle mt-2 flex h-16 w-full flex-row items-center justify-start border-t px-2.5 `}
      >
        <div
          className={`hover:bg-canvasSubtle text-subtle flex w-full flex-row items-center rounded p-1 ${
            collapsed ? 'justify-center' : 'justify-start'
          }`}
        >
          <div className="bg-primary-moderate text-onContrast flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs uppercase">
            <RiCodeSSlashLine className="h-4 w-4" />
          </div>

          {!collapsed && (
            <div className="ml-2 flex flex-col items-start justify-start overflow-hidden">
              <div className="text-subtle leading-1 max-w-full text-sm">
                Settings
              </div>
              <div className="text-muted max-w-full text-xs leading-4">
                Dev Server
              </div>
            </div>
          )}
        </div>
      </div>
    </ProfileMenu>
  );
};

export default Profile;
