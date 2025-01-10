'use client';

import { useRef } from 'react';
import { Button } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { RiRefreshLine } from '@remixicon/react';

import { Runs } from '@/components/Runs';
import type { RefreshRunsRef } from '@/components/Runs/Runs';

export default function Page() {
  const ref = useRef<RefreshRunsRef>(null);
  return (
    <>
      <Header
        breadcrumb={[{ text: 'Runs' }]}
        action={
          <Button
            kind="primary"
            appearance="outlined"
            label="Refresh runs"
            icon={<RiRefreshLine />}
            iconSide="left"
            onClick={() => ref.current?.refresh()}
          />
        }
      />
      <Runs scope="env" ref={ref} />
    </>
  );
}
