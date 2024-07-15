import type { ComponentType } from 'react';
import type { Route } from 'next';
import Image from 'next/image';
import Link from 'next/link';
import { auth, clerkClient } from '@clerk/nextjs';
import type { Organization, OrganizationMembership } from '@clerk/nextjs/server';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import {
  RiAddCircleLine,
  RiArrowLeftRightLine,
  RiBankCardLine,
  RiBox3Line,
  RiSettings3Line,
  RiTeamFill,
} from '@remixicon/react';

export default async function OrganizationDropdown() {
  const { orgId: organizationId } = auth();

  if (!organizationId) {
    return null;
  }

  const organizations = (
    await clerkClient.organizations.getOrganizationMembershipList({
      organizationId,
    })
  ).map((o: OrganizationMembership) => o.organization);

  const organization = organizations.find((o: Organization) => o.id === organizationId);

  if (!organization) {
    return null;
  }

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
            icon={RiTeamFill}
            href="/settings/organization"
            label="Members"
          />
          <OrganizationDropdownMenuItem
            icon={RiBox3Line}
            href="/settings/integrations"
            label="Integrations"
          />
          <OrganizationDropdownMenuItem
            icon={RiBankCardLine}
            href="/settings/billing"
            label="Billing"
          />
        </DropdownMenuGroup>
        <DropdownMenuGroup className="p-2">
          {organizations.length > 1 ? (
            <DropdownMenuItem
              asChild
              className="p-2 font-medium text-slate-400 outline-none hover:bg-transparent focus:text-white"
            >
              <a href="/organization-list">
                <RiArrowLeftRightLine className="size-4" />
                Switch Organization
              </a>
            </DropdownMenuItem>
          ) : (
            <OrganizationDropdownMenuItem
              icon={RiAddCircleLine}
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
      <Link href={props.href as Route} prefetch={true}>
        <props.icon className="size-4" />
        {props.label}
      </Link>
    </DropdownMenuItem>
  );
}
