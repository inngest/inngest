import type { ReactNode } from 'react';
import type { Route } from 'next';
import Link from 'next/link';

import InngestLogo from '@/icons/InngestLogo';

export const EnvLayout = ({ children }: { children: ReactNode }) => (
  <div className="flex w-full flex-col justify-start">
    <div className="border-subtle flex h-[52px] w-full flex-row items-center justify-start border-b px-6">
      <Link href={process.env.NEXT_PUBLIC_HOME_PATH as Route}>
        <InngestLogo className="text-basis mr-3" width={92} />
      </Link>
      <div className="text-disabled mx-4">|</div>
      <div className="text-basis">All Environments</div>
    </div>
    {children}
  </div>
);
