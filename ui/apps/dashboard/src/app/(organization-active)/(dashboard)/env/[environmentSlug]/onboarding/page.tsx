import { ServerFeatureFlag } from '@/components/FeatureFlags/ServerFeatureFlag';

export default function Onboarding() {
  return (
    <ServerFeatureFlag flag="onboarding-flow-cloud">
      <div>New onboarding page</div>
    </ServerFeatureFlag>
  );
}
