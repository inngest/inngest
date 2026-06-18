import { useState } from 'react';
import {
  Popover,
  PopoverClose,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiArrowDownSLine,
  RiArrowRightSLine,
  RiKey2Line,
} from '@remixicon/react';
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
  const onKeysPage = location.href.startsWith(keysPath);
  const onSigningPage = location.href.startsWith(signingPath);
  const active = onKeysPage || onSigningPage;

  // Collapsed: keep the existing popover-to-the-right behavior.
  if (collapsed) {
    return (
      <Popover>
        <OptionalTooltip tooltip="Keys">
          <PopoverTrigger asChild>
            <button
              type="button"
              className={cn(
                'my-0.5 mx-auto flex h-8 w-8 items-center justify-center rounded',
                active
                  ? 'bg-canvasSubtle text-basis'
                  : 'hover:bg-canvasSubtle text-muted',
              )}
            >
              <RiKey2Line className="h-[16px] w-[16px]" />
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
                Signing key
              </div>
            </Link>
          </PopoverClose>
        </PopoverContent>
      </Popover>
    );
  }

  return (
    <KeysAccordion
      active={active}
      keysPath={keysPath}
      signingPath={signingPath}
      onKeysPage={onKeysPage}
      onSigningPage={onSigningPage}
    />
  );
}

function KeysAccordion({
  active,
  keysPath,
  signingPath,
  onKeysPage,
  onSigningPage,
}: {
  active: boolean;
  keysPath: string;
  signingPath: string;
  onKeysPage: boolean;
  onSigningPage: boolean;
}) {
  // Default to open when a child page is active, so the user lands with the
  // current selection already visible.
  const [open, setOpen] = useState(active);

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className={cn(
          'my-0.5 flex h-7 w-full flex-row items-center gap-2 self-stretch rounded px-2 text-left',
          'hover:bg-canvasSubtle text-muted',
        )}
      >
        <span className="flex shrink-0">
          <RiKey2Line className="h-[16px] w-[16px]" />
        </span>
        <span className="text-sm leading-tight">Keys</span>
        <span className="ml-auto flex shrink-0">
          {open ? (
            <RiArrowDownSLine className="h-4 w-4" />
          ) : (
            <RiArrowRightSLine className="h-4 w-4" />
          )}
        </span>
      </button>
      {open && (
        <div className="border-subtle ml-[11px] flex flex-col border-l pl-3">
          <Link
            to={signingPath}
            className={cn(
              'my-0.5 flex h-7 items-center rounded px-1 text-sm leading-tight',
              onSigningPage
                ? 'bg-canvasSubtle text-basis'
                : 'hover:bg-canvasSubtle text-muted',
            )}
          >
            Signing key
          </Link>
          <Link
            to={keysPath}
            className={cn(
              'my-0.5 flex h-7 items-center rounded px-1 text-sm leading-tight',
              onKeysPage
                ? 'bg-canvasSubtle text-basis'
                : 'hover:bg-canvasSubtle text-muted',
            )}
          >
            Event key
          </Link>
        </div>
      )}
    </>
  );
}
