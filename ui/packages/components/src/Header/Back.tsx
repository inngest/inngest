'use client';

import { useRouter } from 'next/navigation';
import { NewButton } from '@inngest/components/Button';
import { RiArrowLeftLine } from '@remixicon/react';

export const Back = ({ className }: { className?: string }) => {
  const router = useRouter();
  return (
    <NewButton
      kind="secondary"
      appearance="outlined"
      size="small"
      icon={<RiArrowLeftLine />}
      className={className}
      onClick={() => router.back()}
    />
  );
};
