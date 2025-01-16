'use client';

import { useMemo } from 'react';
import { ContextualBanner } from '@inngest/components/Banner';
import { Button } from '@inngest/components/Button';

import { type EntitlementUsageQuery } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { useBooleanLocalStorage } from './localStorage';
import { parseEntitlementUsage } from './parse';

export function BillingBannerView({
  entitlementUsage,
}: {
  entitlementUsage: EntitlementUsageQuery['account']['entitlements'];
}) {
  const { bannerMessage, bannerSeverity, items } = parseEntitlementUsage(entitlementUsage);

  const isVisible = useBooleanLocalStorage('BillingBanner:visible', true);

  const onDismiss = useMemo(() => {
    // Error banners can't be dismissed.
    if (bannerSeverity === 'error') {
      return;
    }

    return () => {
      isVisible.set(false);
    };
  }, [bannerSeverity, isVisible]);

  // Error banners are always visible.
  if (!isVisible.value && bannerSeverity !== 'error') {
    return null;
  }

  if (items.length === 0) {
    return null;
  }

  // Wait for localStorage to be hydrated before rendering the banner.
  if (!isVisible.isReady) {
    return null;
  }

  return (
    <ContextualBanner
      className="flex"
      severity={bannerSeverity}
      onDismiss={onDismiss}
      title={bannerMessage}
      cta={
        <Button
          appearance="outlined"
          href={pathCreator.billing({ tab: 'plans', ref: 'app-billing-banner' })}
          kind="secondary"
          label="Upgrade plan"
        />
      }
    >
      <div className="flex grow">
        <div className="grow">
          <ContextualBanner.List>
            {items.map(([k, v]) => (
              <li key={k}>{v}</li>
            ))}
          </ContextualBanner.List>
        </div>
      </div>
    </ContextualBanner>
  );
}
