'use client';

import dynamic from 'next/dynamic';
import Image from 'next/image';
import NextLink from 'next/link';
import { SignOutButton, useClerk } from '@clerk/nextjs';
import { Listbox } from '@headlessui/react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import {
  RiAddCircleFill,
  RiArrowLeftRightLine,
  RiArrowRightLine,
  RiBillLine,
  RiEqualizerLine,
  RiGroupLine,
  RiLogoutCircleLine,
  RiMore2Line,
  RiUserLine,
} from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

const ModeSwitch = dynamic(() => import('@inngest/components/ThemeMode/ModeSwitch'), {
  ssr: false,
});

type Props = React.PropsWithChildren<{
  isMarketplace: boolean;
}>;

export const ProfileMenu = ({ children, isMarketplace }: Props) => {
  const { client, user, organization, session: currentSession, setActive } = useClerk();
  return (
    <Listbox>
      <Listbox.Button className="w-full cursor-pointer ring-0">{children}</Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase border-muted shadow-primary absolute -right-48 bottom-4 z-50 ml-8 w-max rounded border ring-0 focus:outline-none">
          <Listbox.Option
            disabled
            value="themeMode"
            className="text-muted mx-2 my-2 flex h-8 items-center justify-between px-2 text-[13px]"
          >
            <div>Theme</div>
            <ModeSwitch />
          </Listbox.Option>

          <hr className="border-subtle" />

          <NextLink href="/settings/organization" scroll={false}>
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="org"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiEqualizerLine className="text-muted mr-2 h-4 w-4 " />
                <div>Your Organization</div>
              </div>
            </Listbox.Option>
          </NextLink>
          <NextLink href="/settings/organization/organization-members" scroll={false}>
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="members"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiGroupLine className="text-muted mr-2 h-4 w-4" />
                <div>Members</div>
              </div>
            </Listbox.Option>
          </NextLink>

          {!isMarketplace && (
            <NextLink href={pathCreator.billing()} scroll={false}>
              <Listbox.Option
                className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
                value="billing"
              >
                <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                  <RiBillLine className="text-muted mr-2 h-4 w-4" />
                  <div>Billing</div>
                </div>
              </Listbox.Option>
            </NextLink>
          )}
          <a href="/organization-list">
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle m-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="switchOrg"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiArrowLeftRightLine className="text-muted mr-2 h-4 w-4" />
                <div>Switch Organization</div>
              </div>
            </Listbox.Option>
          </a>

          {/* <hr className="border-subtle" />

          <NextLink href="/settings/user" scroll={false}>
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="userProfile"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiUserLine className="text-muted mr-2 h-4 w-4" />
                <div>Your Profile</div>
              </div>
            </Listbox.Option>
          </NextLink> */}

          <hr className="border-subtle" />

          {currentSession && (
            <Listbox.Option
              className="text-muted m-2 mx-2 flex h-full items-center px-2 text-[13px]"
              value="currentUser"
            >
              <div className="flex w-full flex-col">
                <div className="flex flex-row items-center justify-start">
                  <div>
                    {organization?.hasImage ? (
                      <Image
                        src={organization.imageUrl}
                        className="mr-2 h-4 w-4 rounded-full"
                        alt="Organization Image"
                        width={16}
                        height={16}
                      />
                    ) : (
                      <RiUserLine className="text-muted mr-2 h-4 w-4 rounded-full" />
                    )}
                  </div>
                  <div className="flex flex-col">
                    <div>{user?.fullName}</div>
                    <div>{user?.emailAddresses[0]?.emailAddress}</div>
                  </div>
                </div>
                <div className="flex w-full flex-row justify-between pl-5 pt-2">
                  <NextLink href="/settings/user" scroll={false}>
                    <div className="hover:bg-canvasSubtle flex cursor-pointer flex-row items-center justify-start">
                      <RiUserLine className="text-muted mr-0 h-4 w-4" /> Profile
                    </div>
                  </NextLink>
                  <div className="hover:bg-canvasSubtle flex cursor-pointer flex-row items-center justify-start">
                    <SignOutButton>
                      <div className="hover:bg-canvasSubtle flex cursor-pointer flex-row items-center justify-start">
                        <RiLogoutCircleLine className="text-muted mr-0 h-4 w-4" /> Sign Out
                      </div>
                    </SignOutButton>
                  </div>
                </div>
              </div>
            </Listbox.Option>
          )}

          {client &&
            client.sessions &&
            client.sessions
              .filter((session) => session.id !== currentSession?.id)
              .map((session) => {
                return (
                  <Listbox.Option
                    className="text-muted hover:bg-canvasSubtle m-2 flex h-full cursor-pointer items-center justify-between p-2 text-[13px]"
                    value="currentUser"
                    key={session.id}
                    onClick={() => {
                      setActive({ session: session.id });
                    }}
                  >
                    <div className="flex w-full flex-row items-center">
                      {session.user?.hasImage ? (
                        <div className="mr-0">
                          <Image
                            src={session.user.imageUrl}
                            className="mr-2 h-4 w-4 rounded-full"
                            alt="User profile Image"
                            width={16}
                            height={16}
                          />
                        </div>
                      ) : (
                        <div className="mr-0">
                          <RiUserLine className="h-4 w-4" />
                        </div>
                      )}
                      <div className="flex flex-col">
                        <div>
                          {session.user?.fullName || session.user?.emailAddresses[0]?.emailAddress}
                        </div>
                        <div>{session.user?.emailAddresses[0]?.emailAddress}</div>
                      </div>
                    </div>
                    <RiArrowRightLine className="text-muted h-4 w-4" />
                  </Listbox.Option>
                );
              })}
          <Listbox.Option
            className="text-muted hover:bg-canvasSubtle m-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
            value="addAccount"
          >
            <NextLink href="/sign-in" scroll={false}>
              <div className="hover:bg-canvasSubtle flex cursor-pointer flex-row items-center justify-start">
                <RiAddCircleFill className="text-muted mr-2 h-4 w-4" />
                <div>Add Account</div>
              </div>
            </NextLink>
          </Listbox.Option>

          <hr className="border-subtle" />

          {client.sessions.length > 1 && (
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle m-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="signOut"
            >
              <SignOut isMarketplace={isMarketplace} multiSession={client?.sessions?.length > 1} />
            </Listbox.Option>
          )}
        </Listbox.Options>
      </div>
    </Listbox>
  );
};

function SignOut({
  isMarketplace,
  multiSession,
}: {
  isMarketplace: boolean;
  multiSession: boolean;
}) {
  const content = (
    <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
      <RiLogoutCircleLine className="text-muted mr-2 h-4 w-4" />
      <div>Sign Out{multiSession ? ' Of All Accounts' : ''}</div>
    </div>
  );

  if (!isMarketplace) {
    // Sign out via Clerk.
    return <SignOutButton>{content}</SignOutButton>;
  }

  // Sign out via our backend.
  return <NextLink href={`${process.env.NEXT_PUBLIC_API_URL}/v1/logout`}>{content}</NextLink>;
}
