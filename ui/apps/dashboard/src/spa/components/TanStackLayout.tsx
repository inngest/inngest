import { useOrganization, useUser } from '@clerk/tanstack-react-start';

// import SideBar from '@/components/Layout/SideBar';
import { useEnvironmentContext } from '../contexts/EnvironmentContext';
import TanStackAwareSideBar from './TanStackAwareSideBar';

type TanStackLayoutProps = {
  children: React.ReactNode;
};

export default function TanStackLayout({ children }: TanStackLayoutProps) {
  const { user } = useUser();
  const { organization } = useOrganization();

  let activeEnv: any = undefined;

  try {
    const envContext = useEnvironmentContext();
    activeEnv = envContext.environment;
  } catch (error) {
    // This is normal for non-environment routes (home, about, etc.)
  }

  // Create profile object matching ProfileDisplayType from profile.ts
  const profile = user
    ? {
        isMarketplace: false, // TanStack Router users are not marketplace users
        orgName: organization?.name,
        displayName: user.fullName || user.firstName || user.username || 'User',
        orgProfilePic: organization?.hasImage ? organization.imageUrl : null,
      }
    : null;

  if (!profile) {
    return (
      <div className="flex h-screen w-full items-center justify-center">
        <div className="text-lg">Loading...</div>
      </div>
    );
  }

  return (
    <div className="fixed z-50 flex h-screen w-full flex-row justify-start overflow-y-scroll overscroll-y-none">
      {/* <SideBar activeEnv={activeEnv} collapsed={undefined} profile={profile} /> */}
      <TanStackAwareSideBar activeEnv={activeEnv} collapsed={undefined} profile={profile} />

      <div className="no-scrollbar flex w-full flex-col overflow-x-scroll">{children}</div>
    </div>
  );
}
