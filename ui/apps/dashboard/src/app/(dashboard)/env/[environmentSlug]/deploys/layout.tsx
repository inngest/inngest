import { RocketLaunchIcon } from '@heroicons/react/20/solid';

import Header from '@/components/Header/Header';
import DeployButton from './DeployButton';
import DeployList from './DeployList';

type DeploysLayoutProps = {
  children: React.ReactNode;
};

export default async function DeploysLayout({ children }: DeploysLayoutProps) {
  return (
    <>
      <Header
        title="Deploys"
        icon={<RocketLaunchIcon className="h-3.5 w-3.5 text-white" />}
        action={<DeployButton />}
      />
      <div className="flex flex-grow overflow-hidden bg-slate-50">
        <DeployList />
        {children}
      </div>
    </>
  );
}
