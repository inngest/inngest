'use client';

import { useRef } from 'react';
import { NewButton } from '@inngest/components/Button';
import { RiRefreshLine } from '@remixicon/react';

import { Header } from '@/components/Header/Header';
import { Runs } from '@/components/Runs';
import type { RefreshRunsRef } from '@/components/Runs/Runs';

type RunsProps = {
  params: {
    environmentSlug: string;
  };
};

export default function Page({ params: { environmentSlug: envSlug } }: RunsProps) {
  const ref = useRef<RefreshRunsRef>(null);
  return (
    <>
      <Header
        breadcrumb={[{ text: 'Runs' }]}
        action={
          <NewButton
            kind="primary"
            appearance="outlined"
            label="Refresh runs"
            icon={<RiRefreshLine />}
            iconSide="left"
            onClick={() => ref.current?.refresh()}
          />
        }
      />
      <Runs scope="env" ref={ref} />;
    </>
  );
}
