import type { ReactNode } from 'react';

import { URQLProvider } from '@/queries/URQLProvider';

type OrganizationActiveLayoutProps = {
  children: ReactNode;
};

export default function OrganizationActiveLayout({ children }: OrganizationActiveLayoutProps) {
  return <URQLProvider>{children}</URQLProvider>;
}
