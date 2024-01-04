'use client';

import { type Route } from 'next';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import ArrowPathIcon from '@heroicons/react/20/solid/ArrowPathIcon';
import Squares2X2Icon from '@heroicons/react/20/solid/Squares2X2Icon';
import { Button } from '@inngest/components/Button';
import { Link as InngestLink, defaultLinkStyles } from '@inngest/components/Link';
import * as Tabs from '@radix-ui/react-tabs';

import Input from '@/components/Forms/Input';
import Header from '@/components/Header/Header';
import DeploySigningKey from '../../deploys/DeploySigningKey';

type Props = {
  params: {
    environmentSlug: string;
  };
};

export default function Page({ params: { environmentSlug } }: Props) {
  const router = useRouter();
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
            <p>Provide the location of your App.</p>
          </header>

          <Tabs.Root defaultValue="tab1">
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
                Vercel Sync
              </Tabs.Trigger>
              <Tabs.Trigger
                className="px-4 hover:text-white data-[state=active]:text-white"
                value="tab3"
              >
                Curl Command
              </Tabs.Trigger>
            </Tabs.List>
            <Tabs.Content className="bg-white" value="tab1">
              <div className="border-b border-slate-200 p-8">
                <p>
                  Inngest allows you to host your app code on any platform then invokes your
                  functions via HTTP. To deploy functions to Inngest, all you need to do is tell
                  Inngest where to find them!
                </p>
                <br />
                <p>
                  Since your code is hosted on another platform, you need to register where your
                  functions are hosted with Inngest. For example, if you set up the serve handler (
                  <Link
                    className={defaultLinkStyles}
                    href="https://www.inngest.com/docs/reference/serve#how-the-serve-api-handler-works"
                  >
                    see docs
                  </Link>
                  ) at /api/inngest, and your domain is https://myapp.com, you&apos;ll need to
                  inform Inngest that your app is hosted at https://myapp.com/api/inngest.
                </p>
                <br />
                <p>
                  After you&apos;ve set up the serve API and deployed your application, enter the
                  URL of your application&apos;s serve endpoint to register your functions with
                  Inngest.
                </p>
                <DeploySigningKey className="py-6" />
                <div className="border-t border-slate-200 pt-2">
                  <label htmlFor="url" className="text-slate-500">
                    App URL
                  </label>
                  <Input placeholder="https://example.com/api/inngest" name="url" />
                </div>
              </div>
              <footer className="flex items-center justify-between px-8 py-6">
                {/* To do:  create apps docs and link them here */}
                <InngestLink href="https://www.inngest.com/docs/">View Docs</InngestLink>
                <div className="flex items-center gap-3">
                  <Button
                    label="Cancel"
                    btnAction={() => {
                      router.push(`/env/${environmentSlug}/apps` as Route);
                    }}
                    appearance="outlined"
                  />
                  <Button label="Sync App" btnAction={() => {}} kind="primary" />
                </div>
              </footer>
            </Tabs.Content>
            <Tabs.Content value="tab2"></Tabs.Content>
            <Tabs.Content value="tab3"></Tabs.Content>
          </Tabs.Root>
        </section>
      </div>
    </>
  );
}
