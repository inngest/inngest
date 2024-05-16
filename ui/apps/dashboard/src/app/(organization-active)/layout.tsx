import type { ReactNode } from 'react';
import { Link } from '@inngest/components/Link';

import { Banner } from '@/components/Banner';
import { URQLProvider } from '@/queries/URQLProvider';

type OrganizationActiveLayoutProps = {
  children: ReactNode;
};

export default function OrganizationActiveLayout({ children }: OrganizationActiveLayoutProps) {
  return (
    <URQLProvider>
      <Banner kind="warning">
        We are experiencing some API issues. Please check the{' '}
        <span style={{ display: 'inline-flex' }}>
          <Link href="https://status.inngest.com/">status page.</Link>
        </span>
      </Banner>
      {children}
    </URQLProvider>
  );
}
