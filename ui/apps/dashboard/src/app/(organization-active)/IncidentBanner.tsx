'use client';

import { Banner, type Severity } from '@inngest/components/Banner';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useSystemStatus } from './support/statusPage';

export default function IncidentBanner() {
  const isIncidentBannerEnabled = useBooleanFlag('incident-banner');
  const status = useSystemStatus();

  if (!isIncidentBannerEnabled.value) return;

  let message = 'We are experiencing some issues.';
  let severity: Severity = 'warning';

  if (status.indicator !== 'none') {
    message = `${status.description} -`;
    if (status.indicator === 'minor') {
      severity = 'info';
    }
  }

  return (
    <Banner severity={severity}>
      {message} Please check the{' '}
      <span style={{ display: 'inline-flex' }}>
        <Banner.Link severity={severity} href="https://status.inngest.com/">
          status page
        </Banner.Link>
      </span>
    </Banner>
  );
}
