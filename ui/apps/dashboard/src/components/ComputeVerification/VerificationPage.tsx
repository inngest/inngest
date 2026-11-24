import { useLocation } from '@tanstack/react-router';
import type { ProfileDisplayType } from '@/queries/server/profile';
import TopBar from '../Layout/TopBar';
import BottomBar from '../Layout/BottomBar';
import { ComputeVerification, VerificationLayout } from './ComputeVerification';
import type { VerificationProps } from './types';

/** Keep the existing app navigation available throughout verification. */
export function VerificationPage({
  profile,
  showOnboardingWidget,
  ...props
}: VerificationProps & {
  profile: ProfileDisplayType;
  showOnboardingWidget: () => void;
}) {
  const href = useLocation({ select: (location) => location.href });
  return (
    <VerificationLayout
      header={
        <TopBar
          focused
          profile={profile}
          showOnboardingWidget={showOnboardingWidget}
          organizationRedirectUrl={href}
        />
      }
      footer={<BottomBar focused />}
    >
      <ComputeVerification {...props} />
    </VerificationLayout>
  );
}
