import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';

export default function Mange({ collapsed }: { collapsed: boolean }) {
  return (
    <div className={`jusity-center mt-5 flex flex-col`}>
      {collapsed ? (
        <div className="border-subtle mx-auto mb-1 w-6 border-b" />
      ) : (
        <div className="text-disabled leading-4.5 mx-2.5 mb-1 text-xs font-medium">Monitor</div>
      )}
      <MenuItem
        href="/apps"
        collapsed={collapsed}
        text="Apps"
        icon={<AppsIcon className="h-18px w-[18px]" />}
      />

      <MenuItem
        href="/functions"
        collapsed={collapsed}
        text="Functions"
        icon={<FunctionsIcon className="h-18px w-[18px]" />}
      />
    </div>
  );
}
