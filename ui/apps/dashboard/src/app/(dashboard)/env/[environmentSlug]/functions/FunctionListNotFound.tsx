'use client';

import type { Route } from 'next';
import Image from 'next/image';
import { ClipboardIcon, ExclamationTriangleIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import { useCopyToClipboard } from 'react-use';
import { toast } from 'sonner';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import DevServerImage from '@/images/devserver.png';
import VercelLogomark from '@/logos/vercel-logomark.svg';

export default function FunctionListNotFound() {
  const env = useEnvironment();
  const { value: isAppsEnabled } = useBooleanFlag('apps-page');
  const [, copy] = useCopyToClipboard();

  const command = 'npx inngest-cli@latest dev';
  function copyCommand() {
    copy(command);
    toast.message(
      <>
        <ClipboardIcon className="h-3" /> Copied to clipboard!
      </>
    );
  }

  return (
    <div className="h-full w-full overflow-y-scroll py-16">
      <div className="mx-auto flex w-[640px] flex-col gap-4">
        <div className="text-center">
          <h3 className="mb-4 flex items-center justify-center gap-1 rounded-lg border border-indigo-100 bg-indigo-50 py-2.5 text-lg font-semibold text-indigo-500">
            <ExclamationTriangleIcon className="mt-0.5 h-5 w-5 text-indigo-700" />
            <span>
              No Functions <span className="font-medium text-indigo-900">registered in</span>{' '}
              {env.name}
            </span>
          </h3>
        </div>
        <div className="to-slate-940 bg-slate-910 overflow-hidden rounded-lg bg-gradient-to-br from-slate-900 pt-8">
          <div className="translate-x-1/4">
            <Image
              src={DevServerImage}
              className="overflow-hidden rounded shadow"
              alt="Development Server"
            />
          </div>
          <div className="bg-slate-910/20 -mt-48 px-8 py-6 backdrop-blur-sm">
            <h3 className="flex items-center text-xl font-medium text-white">
              <span className="mr-2 inline-flex h-6 w-6  items-center justify-center rounded-full bg-slate-700 text-center text-sm text-white">
                1
              </span>
              Create your functions
            </h3>
            <p className="mt-2 text-sm tracking-wide text-slate-300">
              The best way to get up and running with Inngest is to install our{' '}
              <span className="font-bold text-white">local dev server</span>. The dev server gives
              you a browser interface that helps you to write, test and debug Inngest functions with
              ease.
            </p>
            <div className="mt-4 flex flex-row gap-2 rounded-lg bg-slate-800 px-3 py-2 font-mono text-sm text-white">
              <ClipboardIcon onClick={copyCommand} className="w-3 cursor-pointer" />
              <span>{command}</span>
            </div>
          </div>

          <div className="flex items-center gap-2 border-t border-slate-800/50 px-8 py-4">
            <Button
              target="_blank"
              href={
                'https://www.inngest.com/docs/quick-start?ref=app-onboarding-functions' as Route
              }
              label=" Read the Quick Start Guide"
            />
          </div>
        </div>

        <div className="rounded-lg border border-slate-300 px-8 pt-8">
          <h3 className="flex items-center text-xl font-semibold text-slate-800">
            <span className="mr-2 inline-flex h-6 w-6  items-center justify-center rounded-full bg-slate-800 text-center text-sm text-white">
              2
            </span>
            Register Your Functions
          </h3>
          <p className="mt-2 text-sm font-medium text-slate-500">
            Inngest functions get {isAppsEnabled ? 'synced' : 'deployed'} along side your existing
            application wherever you already host your app. For Inngest to remotely and securely
            invoke your functions via HTTP, you need to register the URL. You can do this manually
            on the {isAppsEnabled ? 'Apps' : 'Deploys'} tab, or automatically with our Vercel
            integration.
          </p>
          <div className="mt-6 flex items-center gap-2 border-t border-slate-100 py-4">
            <Button
              kind="primary"
              href={`/env/${env.slug}/${isAppsEnabled ? 'apps' : 'deploys'}` as Route}
              label={isAppsEnabled ? 'Go To Apps' : 'Go To Deploys'}
            />
            <div className="flex gap-2 border-l border-slate-100 pl-2">
              <Button
                href={
                  'https://www.inngest.com/docs/deploy/vercel?ref=app-onboarding-functions' as Route
                }
                target="_blank"
                rel="noreferrer"
                appearance="outlined"
                label="Vercel Integration"
                icon={<VercelLogomark className="-ml-0.5" />}
              />
              <Button
                appearance="outlined"
                target="_blank"
                href={
                  isAppsEnabled
                    ? 'https://www.inngest.com/docs/apps?ref=app-onboarding-functions'
                    : ('https://www.inngest.com/docs/deploy?ref=app-onboarding-functions' as Route)
                }
                label="Read The Docs"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
