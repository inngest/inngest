'use client';

import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import ArrowPathIcon from '@heroicons/react/20/solid/ArrowPathIcon';
import Squares2X2Icon from '@heroicons/react/20/solid/Squares2X2Icon';
import { Button } from '@inngest/components/Button';
import { Code } from '@inngest/components/Code';
import { CodeLine } from '@inngest/components/CodeLine';
import { Link } from '@inngest/components/Link';
import * as Tabs from '@radix-ui/react-tabs';

import Header from '@/components/Header/Header';
import { setSkipCacheSearchParam } from '@/utils/urls';
import ManualSync from './ManualSync';

type Props = {
  params: {
    environmentSlug: string;
  };
};

export default function Page({ params: { environmentSlug } }: Props) {
  const router = useRouter();
  const APPS_URL = setSkipCacheSearchParam(`/env/${environmentSlug}/apps`) as Route;
  return (
    <>
      <Header title="Apps" icon={<Squares2X2Icon className="h-5 w-5 text-white" />} />

      <div className="h-full overflow-y-auto bg-slate-100">
        <section className="mx-auto mt-16 max-w-screen-md overflow-hidden rounded-lg">
          <header className="bg-slate-940 px-8 pb-3 pt-6 text-white">
            <div className="flex items-center gap-4 pb-4">
              <ArrowPathIcon className="h-6 w-6" />
              <h2 className="text-xl">Sync App</h2>
            </div>
            <p>Provide the location of your app.</p>
          </header>

          <Tabs.Root defaultValue="tab1" className="bg-white">
            <Tabs.List className="bg-slate-900 pb-2 pl-6 pt-4 text-slate-400">
              <Tabs.Trigger
                className="px-4 hover:text-white data-[state=active]:text-white"
                value="tab1"
              >
                Manual Sync
              </Tabs.Trigger>
              <Tabs.Trigger
                className="px-4 hover:text-white data-[state=active]:text-white"
                value="tab2"
              >
                Vercel Integration
              </Tabs.Trigger>
              <Tabs.Trigger
                className="px-4 hover:text-white data-[state=active]:text-white"
                value="tab3"
              >
                Curl Command
              </Tabs.Trigger>
            </Tabs.List>
            <Tabs.Content value="tab1">
              <ManualSync appsURL={APPS_URL} />
            </Tabs.Content>
            <Tabs.Content value="tab2">
              <div className="border-b border-slate-200 p-8">
                <p>
                  To integrate your code hosted on another platform with Inngest, you need to inform
                  Inngest about the location of your app and functions.
                </p>
                <br />
                <p>
                  Inngest enables you to host your apps on Vercel using their serverless functions
                  platform. By using Inngest&apos;s official Vercel integration, your apps will be
                  synced automatically.
                </p>
              </div>
              <div className="flex items-center justify-between px-8 py-6">
                <Link href="https://www.inngest.com/docs/apps/cloud">View Docs</Link>
                <div className="flex items-center gap-3">
                  <Button
                    label="Cancel"
                    btnAction={() => {
                      router.push(APPS_URL);
                    }}
                    appearance="outlined"
                  />
                  <Button
                    label="Go To Vercel Configuration"
                    btnAction={() => {
                      router.push('/settings/integrations/vercel' as Route);
                    }}
                    kind="primary"
                  />
                </div>
              </div>
            </Tabs.Content>
            <Tabs.Content value="tab3">
              <div className="border-b border-slate-200 p-8">
                <p>
                  To integrate your code hosted on another platform with Inngest, you need to inform
                  Inngest about the location of your app and functions.
                </p>
                <br />
                <p>
                  For example, imagine that your <Code>serve()</Code> handler (
                  <Link
                    showIcon={false}
                    href="https://www.inngest.com/docs/reference/serve#how-the-serve-api-handler-works"
                  >
                    see docs
                  </Link>
                  ) is located at /api/inngest, and your domain is myapp.com. In this scenario,
                  you&apos;ll need to inform Inngest that your apps and functions are hosted at
                  https://myapp.com/api/inngest.
                </p>
                <br />
                <p>
                  You can sync from your machine or automate this within a CI/CD pipeline.
                  <span className="font-semibold">
                    {' '}
                    Send a PUT request to your own application&apos;s serve endpoint using the
                    following command:
                  </span>
                </p>
                <CodeLine code="curl -X PUT https://<your-app>.com/api/inngest" className="mt-6" />
              </div>
              <div className="flex items-center justify-between px-8 py-6">
                <Link href="https://www.inngest.com/docs/apps/cloud">View Docs</Link>
                <div className="flex items-center gap-3">
                  <Button
                    label="Done"
                    btnAction={() => {
                      router.push(APPS_URL);
                    }}
                    kind="primary"
                  />
                </div>
              </div>
            </Tabs.Content>
          </Tabs.Root>
        </section>
      </div>
    </>
  );
}
