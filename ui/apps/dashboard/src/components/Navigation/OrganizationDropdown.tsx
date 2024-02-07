'use client';

import type { Route } from 'next';
import Image from 'next/image';
import { useRouter } from 'next/navigation';
import { useOrganization, useOrganizationList } from '@clerk/nextjs';
import {
  ArrowsRightLeftIcon,
  Cog6ToothIcon,
  CreditCardIcon,
  PlusCircleIcon,
  UserGroupIcon,
} from '@heroicons/react/20/solid';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';

export default function OrganizationDropdown() {
  const { isLoaded, organization } = useOrganization();
  const { userMemberships } = useOrganizationList({ userMemberships: true });
  const router = useRouter();

  if (!isLoaded || !organization) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger>
        <Image
          alt={`${organization.name} profile picture`}
          src={organization.imageUrl}
          width={128}
          height={128}
        />{' '}
        {organization.name}
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem>
          <Cog6ToothIcon />
          Organization Settings
        </DropdownMenuItem>
        <DropdownMenuItem>
          <UserGroupIcon />
          Members
        </DropdownMenuItem>
        <DropdownMenuItem>
          <CreditCardIcon />
          Billing
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        {userMemberships.count && userMemberships.count > 1 ? (
          <DropdownMenuItem onSelect={() => router.push('/organization-list' as Route)}>
            <ArrowsRightLeftIcon />
            Switch Organization
          </DropdownMenuItem>
        ) : (
          <DropdownMenuItem onSelect={() => router.push('/create-organization' as Route)}>
            <PlusCircleIcon />
            Create Organization
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
