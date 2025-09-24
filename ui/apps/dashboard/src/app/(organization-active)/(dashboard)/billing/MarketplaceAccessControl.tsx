'use client';

import { usePathname } from 'next/navigation';

import { pathCreator } from '@/utils/urls';

// Whitelist of paths that marketplace users can access
const marketplaceAllowedPaths = ['/usage'] as const;

interface Props {
  isMarketplace: boolean;
}

export default function MarketplaceAccessControl({
  children,
  isMarketplace,
}: React.PropsWithChildren<Props>) {
  const pathname = usePathname();

  if (isMarketplace) {
    const isAllowed = marketplaceAllowedPaths.some((allowedPath) => pathname.endsWith(allowedPath));

    if (!isAllowed) {
      // Redirect to usage page if trying to access non-whitelisted page
      window.location.href = pathCreator.billing({ tab: 'usage' });
      return null;
    }
  }

  return children;
}
