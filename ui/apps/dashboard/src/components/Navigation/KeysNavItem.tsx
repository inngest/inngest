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
              'my-0.5 flex items-center rounded text-left',
              collapsed
                ? 'mx-auto h-8 w-8 justify-center'
                : 'h-7 w-full flex-row gap-2 self-stretch px-1',
              active
                ? 'bg-canvasSubtle text-basis'
                : 'hover:bg-canvasSubtle text-muted',
            )}
          >
            <span className="flex shrink-0">
              <RiKey2Line className="h-[14px] w-[14px]" />
            </span>
            {!collapsed && <span className="text-sm leading-tight">Keys</span>}
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
