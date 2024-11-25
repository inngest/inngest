'use client';

import { useEffect, useState } from 'react';
import { type Route } from 'next';
import { useSearchParams } from 'next/navigation';
import { useAuth } from '@clerk/nextjs';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { RiArrowLeftLine, RiGithubFill } from '@remixicon/react';
import { ThreadStatus, type ThreadPartsFragment } from '@team-plain/typescript-sdk';
import { useQuery } from 'urql';

import { isEnterprisePlan } from '@/components/Billing/Plans/utils';
import { graphql } from '@/gql';
import LoadingIcon from '@/icons/LoadingIcon';
import cn from '@/utils/cn';
import { SupportForm } from './SupportForm';
import { useSystemStatus } from './statusPage';
import { type TicketType } from './ticketOptions';

const GetAccountSupportInfoDocument = graphql(`
  query GetAccountSupportInfo {
    account {
      id
      plan {
        id
        name
        amount
        features
      }
    }
  }
`);

export default function Page() {
  const status = useSystemStatus();
  const { isSignedIn } = useAuth();
  const searchParams = useSearchParams();
  const [{ data, fetching }] = useQuery({
    query: GetAccountSupportInfoDocument,
    pause: !isSignedIn,
  });

  const plan = data?.account.plan;
  const isPaid = (plan?.amount || 0) > 0;
  const isEnterprise = plan ? isEnterprisePlan(plan) : false;
  const preselectedTicketType = searchParams.get('q') as TicketType;

  return (
    <div className="h-full overflow-y-scroll">
      <div className="mx-auto max-w-screen-xl px-6">
        <div className="my-4 inline-block">
          <Button
            href={process.env.NEXT_PUBLIC_HOME_PATH as Route}
            size="small"
            appearance="outlined"
            icon={<RiArrowLeftLine />}
            label={isSignedIn ? 'Back To Dashboard' : 'Sign In To Dashboard'}
          />
        </div>
        <header className="flex items-center justify-between border-b border-slate-200 py-6">
          <h1 className="text-2xl font-semibold">Inngest Support</h1>
          <div className="" title={`Status updated at ${status.updated_at}`}>
            <a
              href={status.url}
              target="_blank"
              className="hover:text-link flex items-center gap-2 rounded bg-slate-50 px-3 py-1.5 text-sm font-medium text-slate-800 hover:bg-slate-100"
            >
              <span
                className={`mx-1 inline-flex h-2.5 w-2.5 rounded-full`}
                style={{ backgroundColor: status.indicatorColor }}
              ></span>
              {status.description}
            </a>
          </div>
        </header>
        <div className="my-8 grid gap-12 md:grid-cols-2">
          <SupportChannel
            title="Create a ticket"
            label="Account required"
            className="min-h-[384px]"
          >
            {fetching ? (
              <div className="mt-4 flex w-full place-content-center">
                <LoadingIcon />
              </div>
            ) : !isSignedIn ? (
              <>
                <p>Sign in or sign up for an account to create a ticket.</p>
                <div className="flex gap-2">
                  <Button
                    kind="primary"
                    href={`${process.env.NEXT_PUBLIC_SIGN_IN_PATH}?ref=support` as Route}
                    label="Sign In"
                  />
                  <Button
                    kind="primary"
                    href={`${process.env.NEXT_PUBLIC_SIGN_UP_PATH}?ref=support` as Route}
                    label="Sign Up"
                  />
                </div>
              </>
            ) : (
              <>
                <SupportForm
                  isEnterprise={isEnterprise}
                  isPaid={isPaid}
                  preselectedTicketType={preselectedTicketType}
                />
                <SupportTickets isSignedIn={isSignedIn} />
              </>
            )}
          </SupportChannel>
          <SupportChannel title="Live chat" label="Enterprise">
            {fetching ? (
              <div className="mt-4 flex w-full place-content-center">
                <LoadingIcon />
              </div>
            ) : isEnterprise ? (
              <p>
                Create a general request support ticket to request a dedicated Slack or Discord
                channel with the Inngest team.
              </p>
            ) : (
              <p>
                Enterprise plans include live chat support including dedicated Slack channel and
                support SLAs. To chat with someone about our enterprise plans,{' '}
                <Link href="https://inngest.com/contact" className="inline-flex">
                  fill out the form here
                </Link>
                .
              </p>
            )}
          </SupportChannel>

          <SupportChannel title="Community">
            <p>
              Chat with other developers and the Inngest team in our{' '}
              <Link href="https://www.inngest.com/discord" className="inline-flex">
                Discord community
              </Link>
              . Search for topics and questions in our{' '}
              <Link
                href="https://discord.com/channels/842170679536517141/1051516534029291581"
                className="inline-flex"
              >
                #help-forum
              </Link>{' '}
              channel or submit your own question.
            </p>
            <Button
              kind="primary"
              href="https://www.inngest.com/discord"
              target="_blank"
              label="Join our Discord"
            />
          </SupportChannel>
          <SupportChannel title="Open Source">
            <p>File an issue in our open source repos on Github:</p>
            <div>
              <p className="mb-2 text-sm font-medium">Inngest CLI + Dev Server</p>
              <Button
                appearance="outlined"
                href="https://github.com/inngest/inngest/issues"
                label="inngest/inngest"
                icon={<RiGithubFill />}
                className="justify-start"
              />
            </div>
            <div>
              <p className="mb-2 text-sm font-medium">SDKs</p>
              <Button
                appearance="outlined"
                href="https://github.com/inngest/inngest-js/issues"
                label="inngest/inngest-js"
                icon={<RiGithubFill />}
              />
            </div>
          </SupportChannel>
        </div>
        <Footer />
      </div>
    </div>
  );
}

function SupportChannel({
  title,
  label,
  className = '',
  children,
}: {
  title: string;
  label?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn('flex flex-col items-start gap-6 leading-7', className)}>
      <h2 className="flex items-center gap-4 text-lg font-semibold">
        {title}
        {label && (
          <span className="inline-flex items-center rounded px-[5px] py-0.5 text-[12px] font-semibold leading-tight text-indigo-500 ring-1 ring-inset ring-indigo-300">
            {label}
          </span>
        )}
      </h2>
      {children}
    </div>
  );
}

function SupportTickets({ isSignedIn }: { isSignedIn?: boolean }) {
  const [isFetchingTickets, setIsFetchingTickets] = useState(false);
  const [tickets, setTickets] = useState<ThreadPartsFragment[]>([]);
  useEffect(
    function () {
      async function fetchTickets() {
        setIsFetchingTickets(true);
        const result = await fetch(`/api/support-tickets`, {
          method: 'GET',
          credentials: 'include',
          redirect: 'error',
        });
        const body = await result.json();
        if (body) {
          setIsFetchingTickets(false);
          setTickets(body.data);
        }
      }
      fetchTickets();
    },
    [isSignedIn]
  );

  return isFetchingTickets ? (
    <LoadingIcon />
  ) : (
    <div className="w-full">
      <h3 className="mb-2 text-base font-semibold">Recent tickets</h3>
      <div className="border-muted grid w-full grid-cols-1 divide-y divide-slate-300 rounded-md border text-sm">
        {tickets.length > 0
          ? tickets.map((ticket) => (
              <div key={ticket.id} className="flex items-center gap-2 px-2 py-2">
                {ticket.status === ThreadStatus.Done ? (
                  <span className="w-[50px] shrink-0 rounded bg-green-50 px-1.5 py-1 text-center text-xs font-medium text-green-800">
                    Closed
                  </span>
                ) : (
                  <span className="w-[50px] shrink-0 rounded bg-sky-50 px-1.5 py-1 text-center text-xs font-medium text-sky-600">
                    Open
                  </span>
                )}
                <span
                  className="grow overflow-hidden text-ellipsis whitespace-nowrap"
                  title={ticket.previewText || ticket.title}
                >
                  {ticket.previewText || ticket.title}
                </span>
                <span className="flex gap-2">
                  {ticket.labels.map((label) => (
                    <span
                      key={label.id}
                      className="whitespace-nowrap rounded bg-slate-50 px-1.5 py-1 text-center text-xs font-medium text-slate-600"
                    >
                      {label.labelType.name}
                    </span>
                  ))}
                </span>
              </div>
            ))
          : 'No open tickets'}
      </div>
    </div>
  );
}

const FOOTER_NAV_ITEMS = [
  { name: 'Documentation', url: 'https://www.inngest.com/docs?ref=support-center' },
  { name: 'Privacy', url: 'https://www.inngest.com/privacy?ref=support-center' },
  { name: 'Terms & Conditions', url: 'https://www.inngest.com/terms?ref=support-center' },
  { name: 'Security', url: 'https://www.inngest.com/security?ref=support-center' },
];

function Footer() {
  return (
    <div className="mt-32 flex flex-col items-center justify-between gap-8 border-t border-slate-200 py-6 text-sm text-slate-600 md:flex-row">
      <div>© {new Date().getFullYear()} Inngest, Inc. All rights reserved.</div>
      <div className="flex flex-row gap-4">
        {FOOTER_NAV_ITEMS.map((i) => (
          <a href={i.url} key={i.name} className="hover:text-indigo-800">
            {i.name}
          </a>
        ))}
      </div>
    </div>
  );
}
