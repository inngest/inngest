'use client';

import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { IconReplay } from '@inngest/components/icons/Replay';
import {
  RiArrowDownSLine,
  RiCloseCircleLine,
  RiFlashlightFill,
  RiPauseCircleLine,
  RiPlayCircleLine,
} from '@remixicon/react';

import { useBooleanFlag } from '../FeatureFlags/hooks';

export type FunctionActions = {
  showCancel: () => void;
  showInvoke: () => void;
  showPause: () => void;
  showReplay: () => void;
  archived?: boolean;
  paused?: boolean;
};

export const ActionsMenu = ({
  showCancel,
  showInvoke,
  showPause,
  showReplay,
  archived,
  paused,
}: FunctionActions) => {
  const { value: cancelEnabled } = useBooleanFlag('bulk-cancellation-ui');

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          kind="primary"
          appearance="solid"
          size="medium"
          icon={
            <RiArrowDownSLine className="transform-90 transition-transform duration-500 group-data-[state=open]:-rotate-180" />
          }
          iconSide="right"
          label="All actions"
          className="group text-sm"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <OptionalTooltip
          tooltip={
            (archived || paused) &&
            `Invoke not available, function is ${archived ? 'archived' : 'paused'}.`
          }
        >
          <DropdownMenuItem onSelect={showInvoke} disabled={archived || paused}>
            <RiFlashlightFill className="h-4 w-4" />
            Invoke
          </DropdownMenuItem>
        </OptionalTooltip>
        <OptionalTooltip tooltip={archived && 'Pause not available, function is archived.'}>
          <DropdownMenuItem onSelect={showPause} disabled={archived}>
            {paused ? (
              <RiPlayCircleLine className="h-4 w-4" />
            ) : (
              <RiPauseCircleLine className="h-4 w-4" />
            )}
            {paused ? 'Resume' : 'Pause'}
          </DropdownMenuItem>
        </OptionalTooltip>
        <OptionalTooltip
          tooltip={
            (archived || paused) &&
            `Replay not available, function is ${archived ? 'archived' : 'paused'}.`
          }
        >
          <DropdownMenuItem onSelect={showReplay} disabled={archived || paused}>
            <IconReplay className="h-4 w-4" />
            Replay
          </DropdownMenuItem>
        </OptionalTooltip>
        {cancelEnabled && (
          <DropdownMenuItem onSelect={showCancel} className="text-error">
            <RiCloseCircleLine className="h-4 w-4" />
            Bulk Cancel
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
