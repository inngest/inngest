import { Link } from '@inngest/components/Link';

import { Banner } from '@/components/Banner';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';

export default function IncidentBanner() {
  const isIncidentBannerEnabled = useBooleanFlag('incident-banner');
  if (!isIncidentBannerEnabled) return;

  return (
    <Banner kind="warning">
      We are experiencing some API issues. Please check the{' '}
      <span style={{ display: 'inline-flex' }}>
        <Link href="https://status.inngest.com/">status page.</Link>
      </span>
    </Banner>
  );
}
