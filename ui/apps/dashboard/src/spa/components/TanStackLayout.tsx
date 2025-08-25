'use client';

import { useUser } from '@clerk/tanstack-react-start';

import SideBar from '@/components/Layout/SideBar';
import { useEnvironmentContext } from '../contexts/EnvironmentContext';

type TanStackLayoutProps = {
  children: React.ReactNode;
};

export default function TanStackLayout({ children }: TanStackLayoutProps) {
  const { user } = useUser();

  let activeEnv: any = undefined;

  try {
    const envContext = useEnvironmentContext();
    activeEnv = envContext.environment;
  } catch (error) {
    // This is normal for non-environment routes (home, about, etc.)
  }

  const profile = user
    ? {
        id: user.id,
        name: user.fullName || user.firstName || 'User',
        email: user.primaryEmailAddress?.emailAddress || '',
        imageURL: user.imageUrl || '',
        isMarketplace: false,
        displayName: user.fullName || user.firstName || 'User',
        orgProfilePic: user.imageUrl || '',
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
      <SideBar activeEnv={activeEnv} collapsed={undefined} profile={profile} />

      <div className="no-scrollbar flex w-full flex-col overflow-x-scroll">{children}</div>
    </div>
  );
}
