import type { ReactNode } from 'react';

import { URQLProvider } from '@/queries/URQLProvider';
import IncidentBanner from './IncidentBanner';

type OrganizationActiveLayoutProps = {
  children: ReactNode;
};

export default function OrganizationActiveLayout({ children }: OrganizationActiveLayoutProps) {
  return (
    <URQLProvider>
      <IncidentBanner />
      {children}
    </URQLProvider>
  );
}
