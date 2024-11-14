'use client';

import { useMemo } from 'react';
import { NewButton } from '@inngest/components/Button';

import type { EntitlementUsageQuery } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { Banner } from '../Banner';
import { useBooleanLocalStorage } from './localStorage';
import { parseEntitlementUsage } from './parse';

type Props = {
  entitlementUsage: EntitlementUsageQuery['account']['entitlementUsage'];
};

export function BillingBannerView({ entitlementUsage }: Props) {
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
    <Banner className="flex" kind={bannerSeverity} onDismiss={onDismiss}>
      <div className="flex grow">
        <div className="grow">
          {bannerMessage}
          <ul className="list-none">
            {items.map(([k, v]) => (
              <li key={k}>{v}</li>
            ))}
          </ul>
        </div>

        <div className="flex items-center">
          <NewButton
            appearance="outlined"
            href={pathCreator.billing()}
            kind="secondary"
            label="Upgrade plan"
          />
        </div>
      </div>
    </Banner>
  );
}
