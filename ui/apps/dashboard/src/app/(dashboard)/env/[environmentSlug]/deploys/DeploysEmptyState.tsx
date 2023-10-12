'use client';

import { useCallback, useState } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { ExclamationTriangleIcon } from '@heroicons/react/20/solid';
import { capitalCase } from 'change-case';
import { useLocalStorage } from 'react-use';

import Button from '@/components/Button';
import SyntaxHighlighter from '@/components/SyntaxHighlighter';
import LoadingIcon from '@/icons/LoadingIcon';
import VercelLogomark from '@/logos/vercel-logomark-dark.svg';
import { useDeploys } from '@/queries/deploys';
import { DeployFailure } from './DeployFailure';
import DeploySigningKey from './DeploySigningKey';
import { deployViaUrl, type RegistrationFailure } from './utils';

type DeploysOnboarding = {
  environmentSlug: string;
};

const BRANCH_PARENT_ENV_SLUG = 'branch';

export default function DeploysOnboarding({ environmentSlug }: DeploysOnboarding) {
  // We fetch deploys using the same hook at DeployList which enables URQL
  // to only send a single request from the client.
  // We load the deploys here to determine if we should show user onboarding
  // or redirect to the deployment
  const [{ data, fetching }, refetch] = useDeploys({ environmentSlug });
  const [failure, setFailure] = useState<RegistrationFailure>();
  const [input = '', setInput] = useLocalStorage('deploymentUrl', '');
  const [isDeploying, setIsDeploying] = useState(false);
  const router = useRouter();

  const onClickDeploy = useCallback(async () => {
    setIsDeploying(true);

    try {
      const failure = await deployViaUrl(input);
      setFailure(failure);

      if (!failure) {
        refetch();
      }
    } catch {
      setFailure({
        errorCode: undefined,
        headers: {},
        statusCode: undefined,
      });
    } finally {
      setIsDeploying(false);
    }
  }, [input, refetch]);

  const environment = environmentSlug.match(/(production|staging)/i)
    ? capitalCase(environmentSlug)
    : environmentSlug;
  const isBranchParent = environmentSlug === BRANCH_PARENT_ENV_SLUG;

  if (fetching) {
    return (
      <div className="flex h-full flex-grow items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  // Redirect to the first deploy if there are any ONLY if it's not the branch parent
  // Branch parents should always show the empty state w/ deploy instructions
  if (!fetching && data?.deploys?.length && !isBranchParent) {
    const firstDeployId = data?.deploys?.[0]?.id;
    router.push(`/env/${environmentSlug}/deploys/${firstDeployId}` as Route);
    return <></>;
  }

  return (
    <div className="h-full flex-grow overflow-y-scroll py-16">
      <div className="mx-auto flex w-[640px] flex-col gap-4">
        {!isBranchParent && (
          <div className="text-center">
            <h3 className="mb-4 flex items-center justify-center gap-1 rounded-lg border border-indigo-100 bg-indigo-50 py-2.5 text-lg font-semibold text-indigo-500">
              <ExclamationTriangleIcon className="mt-0.5 h-5 w-5 text-indigo-700" />
              <span>
                No Functions <span className="font-medium text-indigo-900">registered in</span>{' '}
                {environment}
              </span>
            </h3>
          </div>
        )}
        {isBranchParent && (
          <div className="rounded-lg border border-slate-300 px-8 pb-4 pt-8">
            <h3 className="flex items-center text-xl font-semibold text-slate-800">
              <span className="mr-2 inline-flex h-6 w-6  items-center justify-center rounded-full bg-slate-800 text-center text-sm text-white">
                1
              </span>
              Set up the SDK
              <div className="ml-4 flex items-center rounded-full bg-slate-200 px-3 py-1 text-xs leading-none text-slate-600">
                Added in v1.7.0
              </div>
            </h3>
            <p className="mt-3 text-sm font-medium text-slate-500">
              To create a branch environment, you need v1.7.0 or later of the Inngest SDK. The SDK
              will automatically get your app&apos;s current git branch for these platform preview
              environments:
            </p>
            <p className="my-4 ml-4 text-sm font-medium text-slate-600">
              Vercel, Netlify, Cloudflare Pages, Render & Railway
            </p>
            <p className="mt-3 text-sm font-medium text-slate-500">
              If you don&apos;t use one of the supported platforms, you can set the environment with
              the{' '}
              <code className="inline-flex rounded bg-slate-700 px-1 font-mono text-xs tracking-tight text-white">
                INNGEST_ENV
              </code>{' '}
              environment variable or by passing the{' '}
              <code className="inline-flex rounded bg-slate-700 px-1 font-mono text-xs tracking-tight text-white">
                env
              </code>{' '}
              option to the{' '}
              <code className="inline-flex rounded bg-slate-700 px-1 font-mono text-xs tracking-tight text-white">
                new Inngest()
              </code>{' '}
              constructor in your code.
            </p>
            <div className="my-4 flex flex-row gap-2 rounded-lg bg-slate-800 px-3 py-2">
              <SyntaxHighlighter language="javascript">
                {`new Inngest({ name: 'My App', env: 'feat/my-branch' })`}
              </SyntaxHighlighter>
            </div>
            {/* <div className="mt-6 flex items-center gap-2 border-t border-slate-100 pt-4">
              <Button
                variant="secondary"
                target="_blank"
                href={'https://www.inngest.com/docs/functions' as Route}
              >
                Read The Docs
              </Button>
            </div> */}
          </div>
        )}
        <div className="bg-slate-910 to-slate-910 rounded-lg bg-gradient-to-br from-slate-900 px-8 pt-8">
          <div className="bg-slate-910/20 -mt-8 pt-6 backdrop-blur-sm">
            <h3 className="flex items-center text-xl font-medium text-white">
              <span className="mr-2 inline-flex h-6 w-6  items-center justify-center rounded-full bg-slate-700 text-center text-sm text-white">
                {isBranchParent ? '2' : '1'}
              </span>
              Register your functions
            </h3>
            <p className="mt-2 text-sm tracking-wide text-slate-300">
              Inngest functions get deployed along side your existing application wherever you
              already host your app. Inngest invokes your functions via HTTP so the only thing
              Inngest needs to know is <em>where</em> your functions have been deployed.
            </p>
            <p className="mt-2 text-sm tracking-wide text-slate-300">
              If you&apos;re using Vercel, you can use our{' '}
              <span className="font-bold text-white">Vercel Integration</span> to automatically
              register your functions every time you deploy your app.
            </p>
            <p className="mt-6 border-t border-slate-800/50 pt-6 text-sm tracking-wide text-slate-300">
              You can also manually register your functions by following the steps below:
            </p>
            <h4 className="mt-4 text-base font-semibold text-white">Add Your Signing Key</h4>
            <p className="mb-4 mt-2 text-sm tracking-wide text-slate-300">
              Inngest can invoke your functions remotely and securely with help of the Inngest
              signing key. Set the{' '}
              <code className="inline-flex rounded bg-slate-700 px-1 font-mono text-xs tracking-tight text-white">
                INNGEST_SIGNING_KEY
              </code>{' '}
              environment variable in your application with the value below.
            </p>
            <DeploySigningKey context="dark" environmentSlug={environmentSlug} />
            <h4 className="mt-6 text-base font-semibold text-white">Add your API URL</h4>
            <p className="mt-2 text-sm tracking-wide text-slate-300">
              After you&apos;ve set up the{' '}
              <code className="inline-flex rounded bg-slate-700 px-1 font-mono text-xs tracking-tight text-white">
                serve
              </code>{' '}
              API and deployed your application, enter the URL of your application&apos;s{' '}
              <code className="inline-flex rounded bg-slate-700 px-1 font-mono text-xs tracking-tight text-white">
                serve
              </code>{' '}
              endpoint to register your functions with Inngest.
            </p>
            <input
              type="text"
              className="mt-4 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-white focus:outline-indigo-500"
              placeholder="https://example.com/api/inngest"
              value={input}
              onChange={(e) => setInput(e.target.value)}
            />
          </div>

          <div className="mt-6 flex items-center gap-2 border-t border-slate-800/50 py-4">
            <Button
              variant="primary"
              target="_blank"
              onClick={() => onClickDeploy()}
              disabled={isDeploying || input.length === 0}
            >
              {isDeploying ? 'Deploying your functions...' : 'Deploy Your Functions'}
            </Button>
            <div className="flex gap-2 border-l border-slate-800/50 pl-2">
              <Button
                href={
                  'https://www.inngest.com/docs/deploy/vercel?ref=app-onboarding-deploys' as Route
                }
                target="_blank"
                rel="noreferrer"
                variant="secondary"
                context="dark"
              >
                <VercelLogomark className="-ml-0.5 h-4 w-4" />
                Vercel Integration
              </Button>
              <Button
                variant="secondary"
                context="dark"
                target="_blank"
                href={'https://www.inngest.com/docs/deploy?ref=app-onboarding-deploys' as Route}
              >
                Read The Docs
              </Button>
            </div>
          </div>
          {failure && !isDeploying ? <DeployFailure {...failure} /> : null}
          <div>hi</div>
        </div>

        <div className="rounded-lg border border-slate-300 px-8 pt-8">
          <h3 className="flex items-center text-xl font-semibold text-slate-800">
            <span className="mr-2 inline-flex h-6 w-6  items-center justify-center rounded-full bg-slate-800 text-center text-sm text-white">
              {isBranchParent ? '3' : '2'}
            </span>
            Trigger Functions With Events
          </h3>
          <p className="mt-2 text-sm font-medium text-slate-500">
            After registering your functions, you can trigger them with events sent to this
            environment. {!isBranchParent && 'View the Events tab or read the docs to learn how.'}
          </p>
          <div className="mt-6 flex items-center gap-2 border-t border-slate-100 py-4">
            {isBranchParent ? (
              <Button variant="primary" href={`/env/${environmentSlug}/manage/keys`}>
                Get Event Key
              </Button>
            ) : (
              <Button variant="primary" href={`/env/${environmentSlug}/events` as Route}>
                Go To Events
              </Button>
            )}

            <div className="flex gap-2 border-l border-slate-100 pl-2">
              <Button
                variant="secondary"
                target="_blank"
                href={'https://www.inngest.com/docs/functions' as Route}
              >
                Read The Docs
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
