import { Button } from '@inngest/components/Button';
import {
  RiArrowRightUpLine,
  RiHistoryLine,
  RiTimer2Line,
} from '@remixicon/react';

import { entitlementSecondsToStr } from '@/utils/entitlementTimeFmt';
import { pathCreator } from '@/utils/urls';

type Props = {
  granularitySeconds: number;
  freshnessSeconds: number;
  className?: string;
};

export default function MetricsExportEntitlementBanner({
  granularitySeconds,
  freshnessSeconds,
  className,
}: Props) {
  return (
    <>
      <div
        className={`border-subtle p-3 pt-2 ${className}`}
        style={{
          borderWidth: '1px',
          borderRadius: '4px',
          display: 'flex',
          flexDirection: 'row',
          alignItems: 'center',
        }}
      >
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            flex: '1',
            gap: '0.3rem',
          }}
        >
          <div>
            <span className="text-muted text-xs font-semibold uppercase">
              Your Current Plan
            </span>
          </div>
          <div
            style={{
              display: 'flex',
              flexDirection: 'row',
              gap: '0.3rem',
            }}
          >
            <span className="text-light inline-block">
              <RiTimer2Line
                className="h-4 w-4"
                style={{ marginTop: '0.24rem' }}
              />
            </span>
            <span className="text-muted inline-block">Granularity</span>
            <span
              className="inline-block font-medium"
              style={{ marginLeft: '0.5rem' }}
            >
              {entitlementSecondsToStr(granularitySeconds)}
            </span>
            <span
              className="text-light inline-block"
              style={{ marginLeft: '2.5rem' }}
            >
              <RiHistoryLine
                className="h-4 w-4"
                style={{ marginTop: '0.24rem' }}
              />
            </span>
            <span className="text-muted inline-block">Delay</span>
            <span
              className="inline-block font-medium"
              style={{ marginLeft: '0.5rem' }}
            >
              {entitlementSecondsToStr(freshnessSeconds)}
            </span>
          </div>
        </div>
        <div>
          <Button
            appearance="solid"
            kind="primary"
            label={
              <span>
                <RiArrowRightUpLine className="-ml-0.5 -mt-0.5 mr-0.5 inline h-5 w-5" />
                Customize limits
              </span>
            }
            className="text-sm"
            href={pathCreator.billing({
              tab: 'plans',
              ref: 'metrics-export-entitlements-banner',
            })}
          />
        </div>
      </div>
    </>
  );
}
