import Image from 'next/image';
import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';

import { getProfileDisplay } from '@/queries/server-only/profile';

export default async function SettingsLayout({ children }: { children: React.ReactNode }) {
  const profile = await getProfileDisplay();
  return (
    <div className="h-full overflow-y-scroll">
      <div className="mx-auto flex h-full max-w-screen-xl flex-col px-6">
        <header className="flex items-center justify-between py-6">
          <div>
            <InngestLogo />
            <h1 className="hidden">Inngest</h1>
          </div>
          <div className="flex items-center gap-2">
            <div className="bg-canvasMuted text-subtle flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs uppercase">
              {profile.orgProfilePic ? (
                <Image
                  src={profile.orgProfilePic}
                  className="h-8 w-8 rounded-full object-cover"
                  width={32}
                  height={32}
                  alt="org-profile-pic"
                />
              ) : (
                profile.orgName?.substring(0, 2) || '?'
              )}
            </div>
            <p>{profile.orgName}</p>
          </div>
        </header>
        <div className="flex grow items-center">{children}</div>
      </div>
    </div>
  );
}
