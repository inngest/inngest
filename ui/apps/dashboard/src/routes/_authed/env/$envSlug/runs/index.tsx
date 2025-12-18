import { Button } from '@inngest/components/Button/NewButton';
import { Header } from '@inngest/components/Header/NewHeader';
import { RiRefreshLine } from '@remixicon/react';
import { createFileRoute, ClientOnly } from '@tanstack/react-router';
import { Runs } from '@/components/Runs/Runs';
import { useRef } from 'react';
import { type RefreshRunsRef } from '@/components/Runs/Runs';

export const Route = createFileRoute('/_authed/env/$envSlug/runs/')({
  component: RunsComponent,
});

function RunsComponent() {
  const ref = useRef<RefreshRunsRef>(null);

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Runs' }]}
        action={
          <div className="flex flex-row items-center justify-end gap-2">
            <Button
              kind="primary"
              appearance="outlined"
              label="Refresh runs"
              icon={<RiRefreshLine />}
              iconSide="left"
              onClick={() => ref.current?.refresh()}
            />
          </div>
        }
      />
      <ClientOnly>
        <Runs scope="env" ref={ref} />
      </ClientOnly>
    </>
  );
}
