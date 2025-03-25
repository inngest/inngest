'use client';

import { useRouter } from 'next/navigation';
import { RiRefreshLine } from '@remixicon/react';
import { toast } from 'sonner';

import { Button } from '../Button';

export const RefreshButton = () => {
  const router = useRouter();

  return (
    <Button
      kind="primary"
      appearance="outlined"
      label="Refresh page"
      icon={<RiRefreshLine />}
      iconSide="left"
      onClick={() => {
        router.refresh();
        setTimeout(() => toast.success('Page successfully refreshed!'), 500);
      }}
    />
  );
};
