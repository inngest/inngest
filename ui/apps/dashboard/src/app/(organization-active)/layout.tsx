import type { ReactNode } from 'react';
import { redirect } from 'next/navigation';
import { auth } from '@clerk/nextjs';

import { URQLProvider } from '@/queries/URQLProvider';

type OrganizationActiveLayoutProps = {
  children: ReactNode;
};

export default function OrganizationActiveLayout({ children }: OrganizationActiveLayoutProps) {
  const { userId, orgId } = auth();
  const isSignedIn = !!userId;
  const hasActiveOrganization = !!orgId;

  if (isSignedIn && !hasActiveOrganization) {
    redirect('/organization-list');
  }

  return <URQLProvider>{children}</URQLProvider>;
}
