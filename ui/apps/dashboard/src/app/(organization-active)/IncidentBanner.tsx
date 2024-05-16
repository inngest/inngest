'use client';

import { Link } from '@inngest/components/Link';

import { Banner, type Severity } from '@/components/Banner';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useSystemStatus } from './support/statusPage';

export default function IncidentBanner() {
  const isIncidentBannerEnabled = useBooleanFlag('incident-banner');
  const status = useSystemStatus();

  if (!isIncidentBannerEnabled.value && status.indicator === 'none') return;

  let message = '';
  let severity: Severity = 'warning';

  if (isIncidentBannerEnabled.value) {
    message = ' We are experiencing some issues.';
  } else {
    message = `${status.description}`;
    if (status.indicator === 'minor') {
      severity = 'info';
    }
  }

  return (
    <Banner kind={severity}>
      {message} Please check the{' '}
      <span style={{ display: 'inline-flex' }}>
        <Link href="https://status.inngest.com/">status page.</Link>
      </span>
    </Banner>
  );
}
