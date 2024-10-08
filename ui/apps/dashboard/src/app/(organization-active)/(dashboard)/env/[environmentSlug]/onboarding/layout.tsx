import { Header } from '@inngest/components/Header/Header';

import { ServerFeatureFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { pathCreator } from '@/utils/urls';

export default function Layout({ children }: React.PropsWithChildren) {
  return (
    <ServerFeatureFlag flag="onboarding-flow-cloud">
      <Header breadcrumb={[{ text: 'Getting started', href: pathCreator.onboarding() }]} />
      {children}
    </ServerFeatureFlag>
  );
}
