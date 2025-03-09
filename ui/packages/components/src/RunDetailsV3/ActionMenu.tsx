'use client';

import { useState } from 'react';
import { SplitButton } from '@inngest/components/Button/SplitButton';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import {
  RiArrowDownSLine,
  RiArrowLeftSLine,
  RiArrowRightSLine,
  RiCloseCircleLine,
} from '@remixicon/react';
import { toast } from 'sonner';

import { Button } from '../Button';
import { Link } from '../Link';
import { useRerun } from '../Shared/useRerun';
import { RerunModal } from './RerunModal';

export type RunActions = {
  cancel: () => void;
  allowCancel?: boolean;
  runID: string;
  fnID?: string;
};

export const ActionsMenu = ({ cancel, allowCancel, runID, fnID }: RunActions) => {
  const [menuOpen, setMenuOpen] = useState(false);
  const [rerunOpen, setRerunOpen] = useState(false);
  return (
    <>
      <RerunModal runID={runID} fnID={fnID} open={rerunOpen} onClose={() => setRerunOpen(false)} />

      <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
        <DropdownMenuTrigger className="h-7" />
        <SplitButton
          left={
            <Button
              kind="primary"
              appearance="outlined"
              size="medium"
              label="Rerun"
              onClick={(e) => {
                setRerunOpen(!rerunOpen);
              }}
            />
          }
          right={
            <Button
              kind="primary"
              appearance="outlined"
              size="medium"
              icon={menuOpen ? <RiArrowLeftSLine /> : <RiArrowDownSLine />}
              className="*:transform-90 *:transition-transform *:duration-500"
              onClick={(e) => {
                setMenuOpen(!menuOpen);
              }}
            />
          }
        />

        <DropdownMenuContent align="start">
          <DropdownMenuItem
            onSelect={cancel}
            className={`text-error ${!allowCancel && 'cursor-not-allowed'}`}
            disabled={!allowCancel}
          >
            <RiCloseCircleLine className="h-4 w-4" />
            Cancel
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
};
