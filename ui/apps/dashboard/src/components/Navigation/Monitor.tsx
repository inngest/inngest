import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { EventLogsIcon } from '@inngest/components/icons/sections/EventLogs';
import { InsightsIcon } from '@inngest/components/icons/sections/Insights';
import { MetricsIcon } from '@inngest/components/icons/sections/Metrics';
import { RunsIcon } from '@inngest/components/icons/sections/Runs';

import type { Environment as EnvType } from '@/utils/environments';
import { useBooleanFlag } from '../FeatureFlags/hooks';
import { getNavRoute } from './Navigation';

export default function Monitor({
  activeEnv,
  collapsed,
}: {
  activeEnv: EnvType;
  collapsed: boolean;
}) {
  const { value: isInsightsEnabled } = useBooleanFlag('insights');

  return (
    <div className={`flex w-full flex-col  ${collapsed ? 'mt-2' : 'mt-5'}`}>
      {collapsed ? (
        <hr className="border-subtle mx-auto mb-1 w-6" />
      ) : (
        <div className="text-muted leading-4.5 mb-1 text-xs font-medium">Monitor</div>
      )}
      <MenuItem
        href={getNavRoute(activeEnv, 'metrics')}
        collapsed={collapsed}
        text="Metrics"
        icon={<MetricsIcon className="h-[18px] w-[18px]" />}
      />
      <MenuItem
        href={getNavRoute(activeEnv, 'runs')}
        collapsed={collapsed}
        text="Runs"
        icon={<RunsIcon className="h-[18px] w-[18px]" />}
      />
      <MenuItem
        href={getNavRoute(activeEnv, 'events')}
        collapsed={collapsed}
        text="Events"
        icon={<EventLogsIcon className="h-[18px] w-[18px]" />}
      />
      {isInsightsEnabled && (
        <MenuItem
          href={getNavRoute(activeEnv, 'insights')}
          beta
          collapsed={collapsed}
          text="Insights"
          icon={<InsightsIcon className="h-[18px] w-[18px]" />}
        />
      )}
    </div>
  );
}
