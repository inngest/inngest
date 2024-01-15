import { RocketLaunchIcon } from '@heroicons/react/20/solid';

import { Banner } from '@/components/Banner';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Header from '@/components/Header/Header';
import DeployButton from './DeployButton';
import DeployList from './DeployList';

type DeploysLayoutProps = {
  children: React.ReactNode;
};

export default async function DeploysLayout({ children }: DeploysLayoutProps) {
  const isAppsEnabled = await getBooleanFlag('apps-page');
  return (
    <>
      <Header
        title="Deploys"
        icon={<RocketLaunchIcon className="h-3.5 w-3.5 text-white" />}
        action={<DeployButton />}
      />
      {isAppsEnabled && (
        <Banner kind="error">
          <p className="pr-2 text-red-800">
            The deploys page is getting deprecated. We&apos;ve moved all this information and
            functionality over to the <b>Apps</b> tab.
          </p>
          {/* To do: wire this to the docs */}
          {/* <Link internalNavigation={false} href="">
            Learn More
          </Link> */}
        </Banner>
      )}
      <div className="flex flex-grow overflow-hidden bg-slate-50">
        <DeployList />
        {children}
      </div>
    </>
  );
}
