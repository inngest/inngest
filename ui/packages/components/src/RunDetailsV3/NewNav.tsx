import { RiArrowDownSLine, RiExternalLinkLine, RiShareForward2Line } from '@remixicon/react';
import { useRouter } from '@tanstack/react-router';

import { Button } from '../Button/NewButton';
import { SplitButton } from '../Button/SplitButton';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../DropdownMenu';
import { Link } from '../Link/NewLink';
import { useShared } from '../SharedContext/SharedContext';
import { useBooleanFlag } from '../SharedContext/useBooleanFlag';
import { usePathCreator } from '../SharedContext/usePathCreator';

type NavProps = {
  standalone: boolean;
  functionSlug: string;
  runID: string;
};

export const Standalone = ({ runID }: NavProps) => {
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
          <Link to={pathCreator.runPopout({ runID })} className="flex items-center gap-2">
            <RiShareForward2Line className="h-4 w-4" />
            Open in new tab
          </Link>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

export const Nav = ({ standalone, functionSlug, runID }: NavProps) => {
  const router = useRouter();
  const { booleanFlag } = useBooleanFlag();
  const { pathCreator } = usePathCreator();
  const { cloud } = useShared();

  const { value: debuggerEnabled, isReady: debuggerFlagReady } = booleanFlag(
    'step-over-debugger',
    false
  );

  const debuggerRedirect = async (e: React.MouseEvent) => {
    e?.preventDefault && e.preventDefault();
    const debuggerPath = pathCreator.debugger({
      functionSlug,
      runID,
      debugSessionID: runID,
    });

    router.navigate({ to: debuggerPath });
    return;
  };

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
    </>
  );
};
