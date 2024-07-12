import type { ReactNode } from 'react';
import type { Route } from 'next';
import Image from 'next/image';
import Link from 'next/link';
import { SignOutButton, currentUser } from '@clerk/nextjs';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import {
  RiMapPinLine,
  RiNewspaperLine,
  RiQuestionLine,
  RiSettings3Line,
  RiShutDownLine,
} from '@remixicon/react';

// import type { StatusPageStatusResponse } from '@/app/(organization-active)/support/statusPage';
import SystemStatusIcon from '@/components/Navigation/old/SystemStatusIcon';
import { getStatus } from '../../Support/Status';

export default async function UserDropdown() {
  const user = await currentUser();
  const status = await getStatus();

  if (!user) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex h-full items-center gap-2 border-l border-slate-800 px-2 py-1.5 text-sm tracking-wide text-white hover:bg-slate-800 md:px-4">
        <Image
          alt="Your profile picture"
          src={user.imageUrl}
          width={20}
          height={20}
          className="size-5 rounded-full"
        />
      </DropdownMenuTrigger>

      <DropdownMenuContent
        sideOffset={4}
        className="bg-slate-940/95 z-50 min-w-[200px] divide-y divide-dashed divide-slate-700 p-0 backdrop-blur"
      >
        <DropdownMenuGroup className="p-2">
          <OrganizationDropdownMenuItem
            icon={<RiSettings3Line className="size-4" />}
            href="/settings/user"
            label="User Settings"
          />
        </DropdownMenuGroup>
        <DropdownMenuGroup className="p-2">
          <OrganizationDropdownMenuItem
            icon={<RiMapPinLine className="size-4" />}
            href="https://roadmap.inngest.com/roadmap"
            label="Roadmap"
          />
          <OrganizationDropdownMenuItem
            icon={<RiNewspaperLine className="size-4" />}
            href="https://roadmap.inngest.com/changelog"
            label="Release Notes"
          />
          <OrganizationDropdownMenuItem
            icon={<RiQuestionLine className="size-4" />}
            href="/support"
            label="Contact Support"
          />
          <OrganizationDropdownMenuItem
            icon={<SystemStatusIcon status={status} className="size-4" />}
            href={status.url}
            label="Status Page"
          />
        </DropdownMenuGroup>
        <DropdownMenuGroup className="p-2">
          <SignOutButton>
            <OrganizationDropdownMenuItem
              icon={<RiShutDownLine className="size-4" />}
              label="Sign Out"
            />
          </SignOutButton>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function OrganizationDropdownMenuItem({
  icon,
  label,
  href,
  onSelect,
}: {
  icon: ReactNode;
  label: string;
  href?: string;
  onSelect?: () => void;
}) {
  return (
    <DropdownMenuItem
      onSelect={onSelect}
      asChild
      className="p-2 font-medium text-slate-400 outline-none hover:bg-transparent focus:text-white"
    >
      {href ? (
        <Link href={href as Route}>
          {icon}
          {label}
        </Link>
      ) : (
        <button>
          {icon}
          {label}
        </button>
      )}
    </DropdownMenuItem>
  );
}
