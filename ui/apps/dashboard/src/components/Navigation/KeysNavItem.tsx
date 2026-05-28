import {
  Popover,
  PopoverClose,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiKey2Line } from '@remixicon/react';
import { Link, useLocation } from '@tanstack/react-router';

import type { Environment as EnvType } from '@/utils/environments';
import { pathCreator } from '@/utils/urls';

export default function KeysNavItem({
  activeEnv,
  collapsed,
}: {
  activeEnv: EnvType;
  collapsed: boolean;
}) {
  const location = useLocation();
  const keysPath = pathCreator.keys({ envSlug: activeEnv.slug });
  const signingPath = pathCreator.signingKeys({ envSlug: activeEnv.slug });
  const active =
    location.href.startsWith(keysPath) || location.href.startsWith(signingPath);

  return (
    <Popover>
      <OptionalTooltip tooltip={collapsed ? 'Keys' : ''}>
        <PopoverTrigger asChild>
          <button
            type="button"
            className={cn(
              'my-0.5 flex h-8 w-full flex-row items-center rounded px-1.5 text-left',
              active
                ? 'bg-secondary-3xSubtle text-info hover:bg-secondary-2xSubtle'
                : 'hover:bg-canvasSubtle text-subtle hover:text-basis',
            )}
          >
            <RiKey2Line className="h-[18px] w-[18px]" />
            {!collapsed && (
              <span className="ml-2.5 text-sm leading-tight">Keys</span>
            )}
          </button>
        </PopoverTrigger>
      </OptionalTooltip>
      <PopoverContent
        align="start"
        side="right"
        sideOffset={8}
        className="w-[160px] py-1"
      >
        <PopoverClose asChild>
          <Link to={keysPath} className="block">
            <div className="text-subtle hover:bg-canvasSubtle hover:text-basis mx-1 my-0.5 flex h-8 cursor-pointer items-center rounded px-2 text-sm">
              Event keys
            </div>
          </Link>
        </PopoverClose>
        <PopoverClose asChild>
          <Link to={signingPath} className="block">
            <div className="text-subtle hover:bg-canvasSubtle hover:text-basis mx-1 my-0.5 flex h-8 cursor-pointer items-center rounded px-2 text-sm">
              Signing keys
            </div>
          </Link>
        </PopoverClose>
      </PopoverContent>
    </Popover>
  );
}
