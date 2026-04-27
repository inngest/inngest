import { useMemo } from 'react';
import { Banner } from '@inngest/components/Banner';
import { Button, type ButtonKind } from '@inngest/components/Button';
import { useBooleanLocalStorage } from '@inngest/components/hooks/useBooleanLocalStorage';

import { type ActiveBannersQuery, BannerSeverity } from '@/gql/graphql';
import { isSafeCTAURL } from './safeUrl';

type BannerData = ActiveBannersQuery['account']['activeBanners'][number];

type BannerKind = 'error' | 'info' | 'success' | 'warning';

const severityMap: Record<BannerSeverity, BannerKind> = {
  [BannerSeverity.Error]: 'error',
  [BannerSeverity.Info]: 'info',
  [BannerSeverity.Success]: 'success',
  [BannerSeverity.Warning]: 'warning',
};

// The shared Button component only exposes primary / secondary / danger kinds.
// Mapping warning → danger keeps the CTA color attention-grabbing; a dedicated
// warning kind would be a nicer follow-up but is out of scope here.
const ctaKindForSeverity: Record<BannerKind, ButtonKind> = {
  error: 'danger',
  warning: 'danger',
  success: 'primary',
  info: 'secondary',
};

export function ActiveBannerItem({ banner }: { banner: BannerData }) {
  const severity = severityMap[banner.severity];
  const isVisible = useBooleanLocalStorage(
    `ActiveBanner:visible:${banner.id}`,
    true,
  );

  const onDismiss = useMemo(() => {
    if (!banner.dismissible) return;
    return () => {
      isVisible.set(false);
    };
  }, [banner.dismissible, isVisible]);

  // Wait for localStorage to hydrate before deciding visibility.
  if (!isVisible.isReady) return null;
  if (banner.dismissible && !isVisible.value) return null;

  // Drop the CTA entirely if the URL scheme is not safe. The backend validates
  // the same schemes, but we defend in depth: banners reach all end users.
  const ctaSafe = banner.cta && isSafeCTAURL(banner.cta.url);

  return (
    <Banner severity={severity} onDismiss={onDismiss}>
      {/* Rendered as plain text; do not switch to dangerouslySetInnerHTML. */}
      <span className="block text-left">
        {banner.title && <strong className="mr-1">{banner.title}</strong>}
        {banner.body}
        {ctaSafe && (
          <Button
            appearance="outlined"
            size="small"
            kind={ctaKindForSeverity[severity]}
            href={banner.cta!.url}
            label={banner.cta!.label}
            target="_blank"
            rel="noopener noreferrer"
            className="ml-3 inline-flex align-middle"
          />
        )}
      </span>
    </Banner>
  );
}
