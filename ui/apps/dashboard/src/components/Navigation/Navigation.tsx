import type { Route } from 'next';
import Link from 'next/link';

import InngestLogo from '@/icons/InngestLogo';
import type { Environment } from '@/utils/environments';
import Search from './Search';

type AppNavigationProps = {
  envs: Environment[];
  activeEnv?: Environment;
  envSlug: string;
};

export default async function Navigation() {
  return (
    <div className="flex-start ml-5 mt-5 flex flex-row items-center">
      <Link href={process.env.NEXT_PUBLIC_HOME_PATH as Route}>
        <InngestLogo className="text-basis mr-3" width={92} />
      </Link>
      <Search />
    </div>
  );
}
