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
          icon={<RiArrowDownSLine />}
          iconSide="right"
          label="All actions"
          className="text-sm"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem>
          <OptionalTooltip
            tooltip={
              (archived || paused) &&
              `Invoke not available, function is ${archived ? 'archived' : 'paused'}.`
            }
          >
            <Button
              onClick={showInvoke}
              disabled={archived || paused}
              appearance="ghost"
              kind="secondary"
              size="small"
              icon={<RiFlashlightFill className="h-4 w-4" />}
              iconSide="left"
              label="Invoke"
              className={`text-muted m-0 w-full justify-start text-sm ${
                (archived || paused) && 'cursor-not-allowed'
              }`}
            />
          </OptionalTooltip>
        </DropdownMenuItem>

        <DropdownMenuItem>
          <OptionalTooltip tooltip={archived && 'Pause not available, function is archived.'}>
            <Button
              onClick={showPause}
              disabled={archived}
              appearance="ghost"
              kind="secondary"
              size="small"
              icon={
                paused ? (
                  <RiPlayCircleLine className="h-4 w-4" />
                ) : (
                  <RiPauseCircleLine className="h-4 w-4" />
                )
              }
              iconSide="left"
              label={paused ? 'Resume' : 'Pause'}
              className={`text-muted m-0 w-full justify-start text-sm ${
                archived && 'cursor-not-allowed'
              }`}
            />
          </OptionalTooltip>
        </DropdownMenuItem>
        <DropdownMenuItem>
          <OptionalTooltip
            tooltip={
              (archived || paused) &&
              `Replay not available, function is ${archived ? 'archived' : 'paused'}.`
            }
          >
            <Button
              onClick={showReplay}
              disabled={archived || paused}
              appearance="ghost"
              kind="secondary"
              size="small"
              icon={<IconReplay className="h-4 w-4" />}
              iconSide="left"
              label="Replay"
              className={`text-muted m-0 w-full justify-start text-sm ${
                (archived || paused) && 'cursor-not-allowed'
              }`}
            />
          </OptionalTooltip>
        </DropdownMenuItem>
        {cancelEnabled && (
          <DropdownMenuItem>
            <Button
              onClick={showCancel}
              appearance="ghost"
              kind="danger"
              size="small"
              icon={<RiCloseCircleLine className="h-4 w-4" />}
              iconSide="left"
              label="Bulk Cancel"
              className="m-0 w-full justify-start text-sm"
            />
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
