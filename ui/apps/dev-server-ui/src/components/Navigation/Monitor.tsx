import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { RiMistLine } from '@remixicon/react';

import { IconFeed } from '@/icons/Feed';

export default function Monitor({ collapsed }: { collapsed: boolean }) {
  return (
    <div className={`jusity-center mt-5 flex flex-col`}>
      {collapsed ? (
        <div className="border-subtle mx-auto mb-1 w-6 border-b" />
      ) : (
        <div className="text-muted leading-4.5 mx-2.5 mb-1 text-xs font-medium">Monitor</div>
      )}
      <MenuItem
        href="/runs"
        collapsed={collapsed}
        text="Runs"
        icon={<RiMistLine className="h-[18px] w-[18px]" />}
      />
      <MenuItem
        href="/stream"
        collapsed={collapsed}
        text="Stream"
        icon={<IconFeed className="h-[18px] w-[18px]" />}
      />
    </div>
  );
}
