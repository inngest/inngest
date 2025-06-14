import NextLink from 'next/link';
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
import { useBooleanFlag } from '../SharedContext/useBooleanFlag';
import { usePathCreator } from '../SharedContext/usePathCreator';

type NavProps = {
  standalone: boolean;
  functionSlug: string;
  runID: string;
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

export const Nav = ({ standalone, functionSlug, runID }: NavProps) => {
  const { booleanFlag } = useBooleanFlag();
  const { pathCreator } = usePathCreator();

  const { value: debuggerEnabled, isReady: debuggerFlagReady } = booleanFlag(
    'step-over-debugger',
    false
  );

  return debuggerFlagReady && debuggerEnabled ? (
    <SplitButton
      left={
        <Button
          size="medium"
          kind="primary"
          appearance="outlined"
          label="Open in Debugger"
          className="rounder-r-none border-r-0"
          href={pathCreator.debugger({ functionSlug, runID })}
        />
      }
      right={<Standalone standalone={standalone} functionSlug={functionSlug} runID={runID} />}
    />
  ) : !standalone ? (
    <Link
      size="medium"
      href={pathCreator.runPopout({ runID })}
      iconAfter={<RiExternalLinkLine className="h-4 w-4 shrink-0" />}
    />
  ) : null;
};
