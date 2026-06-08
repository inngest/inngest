import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { ExperimentsIcon } from '@inngest/components/icons/sections/Experiments';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import type { Environment as EnvType } from '@/utils/environments';
import { getNavRoute } from './Navigation';

export default function AI({
  activeEnv,
  collapsed,
}: {
  activeEnv: EnvType;
  collapsed: boolean;
}) {
  return (
    <div className={`flex w-full flex-col ${collapsed ? 'mt-2' : 'mt-4'}`}>
      {collapsed ? (
        <hr className="border-subtle mx-auto mb-1 w-6" />
      ) : (
        <div className="text-muted leading-4.5 mb-1 text-xs font-medium">
          AI
        </div>
      )}
      <MenuItem
        to={getNavRoute(activeEnv, 'experiments')}
        beta
        collapsed={collapsed}
        text="Experiments"
        icon={<ExperimentsIcon className="h-[18px] w-[18px]" />}
      />
    </div>
  );
}
