import { RiArrowDownSLine, RiExternalLinkLine, RiShareForward2Line } from '@remixicon/react';
import type { LinkComponentProps } from '@tanstack/react-router';

import { Button } from '../Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../DropdownMenu';
import { Link } from '../Link';
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
          <Link
            to={pathCreator.runPopout({ runID }) as LinkComponentProps['to']}
            className="flex items-center gap-2"
          >
            <RiShareForward2Line className="h-4 w-4" />
            Open in new tab
          </Link>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

export const Nav = ({ standalone, runID }: NavProps) => {
  const { pathCreator } = usePathCreator();

  if (standalone) {
    return null;
  }

  return (
    <Link
      size="medium"
      href={pathCreator.runPopout({ runID })}
      target="_blank"
      iconAfter={<RiExternalLinkLine className="h-4 w-4 shrink-0" />}
    />
  );
};
