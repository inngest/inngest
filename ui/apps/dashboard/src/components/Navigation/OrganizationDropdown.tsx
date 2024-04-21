'use client';

import type { ComponentType } from 'react';
import type { Route } from 'next';
import Image from 'next/image';
import Link from 'next/link';
import { useOrganization, useOrganizationList } from '@clerk/nextjs';
import { CreditCardIcon, CubeIcon, PlusCircleIcon, UserGroupIcon } from '@heroicons/react/20/solid';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { Skeleton } from '@inngest/components/Skeleton';
import { RiArrowLeftRightLine, RiSettings3Line } from '@remixicon/react';

export default function OrganizationDropdown() {
  const { isLoaded, organization } = useOrganization();
  const { userMemberships } = useOrganizationList({ userMemberships: true });

  if (!isLoaded) {
    return (
      <div className="flex h-full items-center gap-2 border-l border-slate-800 px-2 py-1.5 md:px-4">
        <Skeleton className="block size-5 rounded" />
        <Skeleton className="h-5 w-20" />
      </div>
    );
  }

  if (!organization) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex h-full max-w-[12rem]	items-center gap-2 border-l border-slate-800 px-2 py-1.5 text-sm tracking-wide text-white hover:bg-slate-800 md:px-4">
        <Image
          alt={`${organization.name} profile picture`}
          src={organization.imageUrl}
          width={20}
          height={20}
          className="size-5 rounded"
        />{' '}
        <span className="truncate">{organization.name}</span>
      </DropdownMenuTrigger>

      <DropdownMenuContent
        sideOffset={4}
        className="bg-slate-940/95 z-50 min-w-[200px] divide-y divide-dashed divide-slate-700 p-0 backdrop-blur"
      >
        <DropdownMenuGroup className="p-2">
          <OrganizationDropdownMenuItem
            icon={RiSettings3Line}
            href="/settings/organization/organization-settings"
            label="Organization Settings"
          />
          <OrganizationDropdownMenuItem
            icon={UserGroupIcon}
            href="/settings/organization"
            label="Members"
          />
          <OrganizationDropdownMenuItem
            icon={CubeIcon}
            href="/settings/integrations"
            label="Integrations"
          />
          <OrganizationDropdownMenuItem
            icon={CreditCardIcon}
            href="/settings/billing"
            label="Billing"
          />
        </DropdownMenuGroup>
        <DropdownMenuGroup className="p-2">
          {userMemberships.count && userMemberships.count > 1 ? (
            <OrganizationDropdownMenuItem
              icon={RiArrowLeftRightLine}
              href="/organization-list"
              label="Switch Organization"
            />
          ) : (
            <OrganizationDropdownMenuItem
              icon={PlusCircleIcon}
              href="/create-organization"
              label="Create Organization"
            />
          )}
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
  href: string;
}) {
  return (
    <DropdownMenuItem
      asChild
      className="p-2 font-medium text-slate-400 outline-none hover:bg-transparent focus:text-white"
    >
      <Link href={props.href as Route}>
        <props.icon className="size-4" />
        {props.label}
      </Link>
    </DropdownMenuItem>
  );
}
