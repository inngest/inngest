'use client';

import { Listbox } from '@headlessui/react';
import { NewButton } from '@inngest/components/Button';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { IconReplay } from '@inngest/components/icons/Replay';
import {
  RiArrowDownSLine,
  RiArrowUpSLine,
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
    <Listbox>
      {({ open }) => (
        <>
          <Listbox.Button as="div">
            <NewButton
              kind="primary"
              appearance="solid"
              size="medium"
              icon={open ? <RiArrowUpSLine /> : <RiArrowDownSLine />}
              iconSide="right"
              label="All actions"
              className="text-sm"
            />
          </Listbox.Button>
          <div className="relative">
            <Listbox.Options className="bg-canvasBase absolute right-1 top-5 z-50 w-[170px] gap-y-0.5 rounded border shadow">
              <Listbox.Option
                className="text-muted mx-2 mt-2 flex h-8 cursor-pointer items-center justify-start text-[13px]"
                value="invoke"
              >
                <OptionalTooltip
                  tooltip={
                    (archived || paused) &&
                    `Invoke not available, function is ${archived ? 'archived' : 'paused'}.`
                  }
                >
                  <NewButton
                    onClick={showInvoke}
                    disabled={archived || paused}
                    appearance="ghost"
                    kind="secondary"
                    size="medium"
                    icon={<RiFlashlightFill className="h-4 w-4" />}
                    iconSide="left"
                    label="Invoke"
                    className={`text-muted m-0 w-full justify-start text-sm ${
                      (archived || paused) && 'cursor-not-allowed'
                    }`}
                  />
                </OptionalTooltip>
              </Listbox.Option>

              <Listbox.Option
                className="m-2 flex h-8 cursor-pointer items-center text-[13px]"
                value="pause"
              >
                <OptionalTooltip tooltip={archived && 'Pause not available, function is archived.'}>
                  <NewButton
                    onClick={showPause}
                    disabled={archived}
                    appearance="ghost"
                    kind="secondary"
                    size="medium"
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
              </Listbox.Option>
              <Listbox.Option
                className="m-2 flex h-8 cursor-pointer items-center text-[13px]"
                value="replay"
              >
                <OptionalTooltip
                  tooltip={
                    (archived || paused) &&
                    `Replay not available, function is ${archived ? 'archived' : 'paused'}.`
                  }
                >
                  <NewButton
                    onClick={showReplay}
                    disabled={archived || paused}
                    appearance="ghost"
                    kind="secondary"
                    size="medium"
                    icon={<IconReplay className="h-4 w-4" />}
                    iconSide="left"
                    label="Replay"
                    className={`text-muted m-0 w-full justify-start text-sm ${
                      (archived || paused) && 'cursor-not-allowed'
                    }`}
                  />
                </OptionalTooltip>
              </Listbox.Option>
              {cancelEnabled && (
                <Listbox.Option
                  className="m-2 flex h-8 cursor-pointer items-center text-[13px]"
                  value="cancel"
                >
                  <NewButton
                    onClick={showCancel}
                    appearance="ghost"
                    kind="danger"
                    size="medium"
                    icon={<RiCloseCircleLine className="h-4 w-4" />}
                    iconSide="left"
                    label="Bulk Cancel"
                    className="m-0 w-full justify-start text-sm"
                  />
                </Listbox.Option>
              )}
            </Listbox.Options>
          </div>
        </>
      )}
    </Listbox>
  );
};
