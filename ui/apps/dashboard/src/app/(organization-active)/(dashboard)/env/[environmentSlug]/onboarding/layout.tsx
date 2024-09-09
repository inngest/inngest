import { Header } from '@inngest/components/Header/Header';

import { ServerFeatureFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Menu from '@/components/Onboarding/Menu';
import { pathCreator } from '@/utils/urls';

export default function Layout({ children }: React.PropsWithChildren<{}>) {
  return (
    <ServerFeatureFlag flag="onboarding-flow-cloud">
      <Header breadcrumb={[{ text: 'Getting started', href: pathCreator.onboarding() }]} />

      <div className="my-12 grid grid-cols-3">
        <main className="col-span-2 mx-20">{children}</main>
        <Menu />
      </div>
    </ServerFeatureFlag>
  );
}
