'use client';

import type { ComponentType } from 'react';
import type { Route } from 'next';
import Image from 'next/image';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth, useUser } from '@clerk/nextjs';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { Skeleton } from '@inngest/components/Skeleton';
import {
  RiMapPinLine,
  RiNewspaperLine,
  RiQuestionLine,
  RiSettings3Line,
  RiShutDownLine,
} from '@remixicon/react';

import { useSystemStatus } from '@/app/(organization-active)/support/statusPage';
import SystemStatusIcon from '@/components/Navigation/SystemStatusIcon';

export default function UserDropdown() {
  const { isLoaded, isSignedIn, user } = useUser();
  const { signOut } = useAuth();
  const status = useSystemStatus();
  const router = useRouter();

  if (!isLoaded) {
    return (
      <div className="flex h-full items-center border-l border-slate-800 px-2 py-1.5 md:px-4">
        <Skeleton className="block size-5 rounded-full" />
      </div>
    );
  }

  if (!isSignedIn) return null;

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
            icon={RiSettings3Line}
            href="/settings/user"
            label="User Settings"
          />
        </DropdownMenuGroup>
        <DropdownMenuGroup className="p-2">
          <OrganizationDropdownMenuItem
            icon={RiMapPinLine}
            href="https://roadmap.inngest.com/roadmap"
            label="Roadmap"
          />
          <OrganizationDropdownMenuItem
            icon={RiNewspaperLine}
            href="https://roadmap.inngest.com/changelog"
            label="Release Notes"
          />
          <OrganizationDropdownMenuItem
            icon={RiQuestionLine}
            href="/support"
            label="Contact Support"
          />
          <OrganizationDropdownMenuItem
            icon={SystemStatusIcon}
            href={status.url}
            label="Status Page"
          />
        </DropdownMenuGroup>
        <DropdownMenuGroup className="p-2">
          <OrganizationDropdownMenuItem
            icon={RiShutDownLine}
            onSelect={() =>
              signOut(() =>
                router.push((process.env.NEXT_PUBLIC_SIGN_IN_PATH || '/sign-in') as Route)
              )
            }
            label="Sign Out"
          />
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function OrganizationDropdownMenuItem(props: {
  icon: ComponentType<{
    className?: string;
  }>;
  label: string;
  href?: string;
  onSelect?: () => void;
}) {
  return (
    <DropdownMenuItem
      onSelect={props.onSelect}
      asChild
      className="p-2 font-medium text-slate-400 outline-none hover:bg-transparent focus:text-white"
    >
      {props.href ? (
        <Link href={props.href as Route}>
          <props.icon className="size-4" />
          {props.label}
        </Link>
      ) : (
        <button>
          <props.icon className="size-4" />
          {props.label}
        </button>
      )}
    </DropdownMenuItem>
  );
}
