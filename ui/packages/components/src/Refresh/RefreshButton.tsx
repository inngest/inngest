'use client';

import { useRouter } from 'next/navigation';
import { RiRefreshLine } from '@remixicon/react';

import { NewButton } from '../Button';

export const RefreshButton = () => {
  const router = useRouter();

  return (
    <NewButton
      kind="primary"
      appearance="outlined"
      label="Refresh page"
      icon={<RiRefreshLine />}
      iconSide="left"
      onClick={() => router.refresh()}
    />
  );
};
