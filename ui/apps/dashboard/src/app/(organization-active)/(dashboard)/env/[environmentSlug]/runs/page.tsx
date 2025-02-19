'use client';

import { useRef } from 'react';
import { Button } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { LegacyRunsToggle } from '@inngest/components/RunDetailsV3/LegacyRunsToggle';
import { useLegacyTrace } from '@inngest/components/Shared/useLegacyTrace';
import { RiRefreshLine } from '@remixicon/react';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { Runs } from '@/components/Runs';
import type { RefreshRunsRef } from '@/components/Runs/Runs';

export default function Page() {
  const ref = useRef<RefreshRunsRef>(null);
  const { value: traceAIEnabled, isReady: featureFlagReady } = useBooleanFlag('ai-traces');

  const { enabled: legacyTraceEnabled, ready: legacyTraceReady } = useLegacyTrace();

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Runs' }]}
        action={
          <div className="flex flex-row items-center justify-end gap-2">
            <LegacyRunsToggle traceAIEnabled={featureFlagReady && traceAIEnabled} />
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
      <Runs
        scope="env"
        ref={ref}
        traceAIEnabled={
          featureFlagReady && traceAIEnabled && legacyTraceReady && !legacyTraceEnabled
        }
      />
    </>
  );
}
