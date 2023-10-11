import { type Route } from 'next';
import Image from 'next/image';
import { SignOutButton, currentUser } from '@clerk/nextjs';
import type { User } from '@clerk/nextjs/server';
import {
  CreditCardIcon,
  CubeIcon,
  LifebuoyIcon,
  MapPinIcon,
  NewspaperIcon,
  PowerIcon,
  UserCircleIcon,
  UserGroupIcon,
} from '@heroicons/react/20/solid';

import Dropdown from '../Dropdown/Dropdown';
import DropdownItem from '../Dropdown/DropdownItem';
import StatusPageItem from './StatusPageItem';

export default async function AccountDropdown() {
  const user = await currentUser();
  if (!user) throw new Error('AccountDropdown must be only used in authenticated pages');

  return (
    <Dropdown
      context="nav"
      label={
        <div className="flex items-center">
          <Image
            src={user.imageUrl}
            width={128}
            height={128}
            alt="Your profile picture"
            className="mr-2 h-5 w-5 rounded-full"
          />
          {getDisplayName(user)}
        </div>
      }
    >
      <div className="p-2">
        <DropdownItem context="dark" href={'/settings/account' as Route}>
          <UserCircleIcon className="h-4" />
          Account
        </DropdownItem>
        <DropdownItem context="dark" href="/settings/billing">
          <CreditCardIcon className="h-4" />
          Billing
        </DropdownItem>
        <DropdownItem context="dark" href={'/settings/integrations' as Route}>
          <CubeIcon className="h-4" />
          Integrations
        </DropdownItem>
        <DropdownItem context="dark" href="/settings/team">
          <UserGroupIcon className="h-4" />
          Team Management
        </DropdownItem>
      </div>
      <div className="p-2">
        <DropdownItem
          context="dark"
          href={'https://roadmap.inngest.com/roadmap' as Route}
          target="_blank"
        >
          <MapPinIcon className="h-4" />
          Roadmap
        </DropdownItem>
        <DropdownItem
          context="dark"
          href={'https://roadmap.inngest.com/changelog' as Route}
          target="_blank"
        >
          <NewspaperIcon className="h-4" /> Release Notes
        </DropdownItem>
        <DropdownItem context="dark" href="/support">
          <LifebuoyIcon className="h-4" />
          Contact support
        </DropdownItem>
        <StatusPageItem />
      </div>
      <div className="p-2">
        <DropdownItem context="dark" Component={SignOutButton}>
          <button>
            <PowerIcon className="h-4" />
            Sign Out
          </button>
        </DropdownItem>
      </div>
    </Dropdown>
  );
}

function getDisplayName(user: User): string {
  let out: string = '';

  if (user.firstName) {
    out = user.firstName;
  }
  if (user.lastName) {
    out += ` ${user.lastName}`;
  }
  if (!out) {
    const email = user.emailAddresses.find((e) => {
      return e.id === user.primaryEmailAddressId;
    });

    // There should always be at least 1 email address, but you never know.
    if (email) {
      out = email.emailAddress;
    }
  }

  if (!out) {
    out = 'Unknown';
  }

  return out.trim();
}
