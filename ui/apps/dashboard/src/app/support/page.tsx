'use client';

import { type Route } from 'next';
import { useAuth } from '@clerk/nextjs';
import { ArrowLeftIcon } from '@heroicons/react/20/solid';
import { useQuery } from 'urql';

import AppLink from '@/components/AppLink';
import Button from '@/components/Button';
import { graphql } from '@/gql';
import LoadingIcon from '@/icons/LoadingIcon';
import GitHubIcon from '@/icons/github.svg';
import cn from '@/utils/cn';
import { isEnterprisePlan } from '../(dashboard)/settings/billing/utils';
import { SupportForm } from './SupportForm';
import { useSystemStatus } from './statusPage';

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
  const [{ data, fetching }] = useQuery({
    query: GetAccountSupportInfoDocument,
  });

  const plan = data?.account.plan;
  const isPaid = (plan?.amount || 0) > 0;
  const isEnterprise = plan ? isEnterprisePlan(plan) : false;

  return (
    <div className="h-full overflow-y-scroll">
      <div className="mx-auto max-w-screen-xl px-6">
        <div className="my-4">
          <Button href={process.env.NEXT_PUBLIC_HOME_PATH as Route} size="sm" variant="secondary">
            <ArrowLeftIcon className="h-3" />{' '}
            {isSignedIn ? 'Back to dashboard' : 'Sign in to dashboard'}
          </Button>
        </div>
        <header className="flex items-center justify-between border-b border-slate-200 py-6">
          <h1 className="text-2xl font-semibold">Inngest Support</h1>
          <div className="" title={`Status updated at ${status.updated_at}`}>
            <a
              href={status.url}
              target="_blank"
              className="flex items-center gap-2 rounded bg-slate-50 px-3 py-1.5 text-sm font-medium text-slate-800 hover:bg-slate-100 hover:text-indigo-500"
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
                  <Button href={`${process.env.NEXT_PUBLIC_SIGN_IN_PATH}?ref=support` as Route}>
                    Sign in
                  </Button>
                  <Button href={`${process.env.NEXT_PUBLIC_SIGN_UP_PATH}?ref=support` as Route}>
                    Sign up
                  </Button>
                </div>
              </>
            ) : (
              <SupportForm isEnterprise={isEnterprise} isPaid={isPaid} />
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
                <AppLink href="https://inngest.com/contact" target="_blank" className="text-base">
                  fill out the form here
                </AppLink>
                .
              </p>
            )}
          </SupportChannel>
          <SupportChannel title="Community">
            <p>
              Chat with other developers and the Inngest team in our{' '}
              <AppLink
                href="https://www.inngest.com/discord"
                target="_blank"
                label="Discord community"
                className="text-base"
              />
              . Search for topics and questions in our{' '}
              <AppLink
                href="https://discord.com/channels/842170679536517141/1051516534029291581"
                target="_blank"
                label="#help-forum"
                className="text-base"
              />{' '}
              channel or submit your own question.
            </p>
            <Button href="https://www.inngest.com/discord" target="_blank">
              Join our Discord
            </Button>
          </SupportChannel>
          <SupportChannel title="Open Source">
            <p>File an issue in our open source repos on Github:</p>
            <div>
              <p className="mb-2 text-sm font-medium">Inngest CLI + Dev Server</p>
              <Button href="https://github.com/inngest/inngest/issues" variant="secondary">
                <GitHubIcon className="-ml-0.5 mr-1 h-4 w-4" /> inngest/inngest
              </Button>
            </div>
            <div>
              <p className="mb-2 text-sm font-medium">SDKs</p>

              <Button href="https://github.com/inngest/inngest-js/issues" variant="secondary">
                <GitHubIcon className="-ml-0.5 mr-1 h-4 w-4" /> inngest/inngest-js
              </Button>
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
