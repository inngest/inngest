import type { ProfileDisplayType } from '@/queries/server/profile';
import TopBar from '../Layout/TopBar';
import BottomBar from '../Layout/BottomBar';
import { ComputeVerification, VerificationLayout } from './ComputeVerification';
import type { VerificationProps } from './types';

/** Host beneath existing app providers when the integration owner adds the route.
 * Shell services remain the existing ones; verification itself has no integrations.
 */
export function VerificationPage({
  profile,
  showOnboardingWidget,
  ...props
}: VerificationProps & {
  profile: ProfileDisplayType;
  showOnboardingWidget: () => void;
}) {
  return (
    <VerificationLayout
      header={
        <TopBar
          focused
          profile={profile}
          showOnboardingWidget={showOnboardingWidget}
        />
      }
      footer={<BottomBar focused />}
    >
      <ComputeVerification {...props} />
    </VerificationLayout>
  );
}
