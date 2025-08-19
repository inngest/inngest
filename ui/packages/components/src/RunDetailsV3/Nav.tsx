'use client';

import { useEffect, useState } from 'react';
import NextLink from 'next/link';
import { useRouter } from 'next/navigation';
import { RiArrowDownSLine, RiExternalLinkLine, RiShareForward2Line } from '@remixicon/react';

import { Button } from '../Button';
import { SplitButton } from '../Button/SplitButton';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../DropdownMenu';
import { Link } from '../Link';
import { AlertModal } from '../Modal/AlertModal';
import { useShared } from '../SharedContext/SharedContext';
import { useBooleanFlag } from '../SharedContext/useBooleanFlag';
import { useCreateDebugSession } from '../SharedContext/useCreateDebugSession';
import { usePathCreator } from '../SharedContext/usePathCreator';

type NavProps = {
  standalone: boolean;
  functionSlug: string;
  runID: string;
  debugRunID?: string | null;
  debugSessionID?: string | null;
};

export const Standalone = ({ standalone, runID }: NavProps) => {
  const { pathCreator } = usePathCreator();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          kind="primary"
          appearance="outlined"
          size="medium"
          icon={
            <RiArrowDownSLine className="transform-90 transition-transform duration-500 group-data-[state=open]:-rotate-180" />
          }
          className="group rounded-l-none text-sm"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem>
          <NextLink href={pathCreator.runPopout({ runID })} className="flex items-center gap-2">
            <RiShareForward2Line className="h-4 w-4" />
            Open in new tab
          </NextLink>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

export const Nav = ({ standalone, functionSlug, runID, debugRunID, debugSessionID }: NavProps) => {
  const router = useRouter();
  const { booleanFlag } = useBooleanFlag();
  const { pathCreator } = usePathCreator();
  const { cloud } = useShared();
  const { createDebugSession, data, isSuccess, loading, error } = useCreateDebugSession();
  const [showErrorModal, setShowErrorModal] = useState(false);

  const { value: debuggerEnabled, isReady: debuggerFlagReady } = booleanFlag(
    'step-over-debugger',
    false
  );

  //
  // if we have debug run/session id, use those for linkable sessions
  const debuggerRedirect = async (e: React.MouseEvent) => {
    if (debugRunID && debugSessionID) {
      const debuggerPath = pathCreator.debugger({
        functionSlug,
        runID,
        debugRunID,
        debugSessionID,
      });

      router.push(debuggerPath);
    }

    e?.preventDefault && e.preventDefault();
    createDebugSession({ functionSlug, runID });
  };

  useEffect(() => {
    if (isSuccess && data) {
      const debuggerPath = pathCreator.debugger({
        functionSlug,
        runID,
        debugRunID: data.debugRunID,
        debugSessionID: data.debugSessionID,
      });
      router.push(debuggerPath);
    }
  }, [isSuccess, data]);

  useEffect(() => {
    if (error) {
      setShowErrorModal(true);
    }
  }, [error]);

  return (
    <>
      {!cloud && debuggerFlagReady && debuggerEnabled ? (
        <SplitButton
          left={
            <Button
              size="medium"
              kind="primary"
              appearance="outlined"
              label="Open in Debugger"
              className="rounder-r-none border-r-0"
              onClick={debuggerRedirect}
            />
          }
          right={<Standalone standalone={standalone} functionSlug={functionSlug} runID={runID} />}
        />
      ) : !standalone ? (
        <Link
          size="medium"
          href={pathCreator.runPopout({ runID })}
          target="_blank"
          iconAfter={<RiExternalLinkLine className="h-4 w-4 shrink-0" />}
        />
      ) : null}
      <AlertModal
        isOpen={showErrorModal}
        title={'Error creating debug session'}
        className="w-[600px]"
        onClose={() => setShowErrorModal(false)}
        onSubmit={() => {
          setShowErrorModal(false);
          debuggerRedirect({} as React.MouseEvent);
        }}
        confirmButtonLabel="Try again"
        cancelButtonLabel="Close"
      >
        <div className="px-6 pt-4">
          {error?.message || 'There was a problem creating a debug session'}
        </div>
      </AlertModal>
    </>
  );
};
