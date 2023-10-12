import type { Route } from 'next';
import { capitalCase } from 'change-case';

import DeployStatus from '@/components/Status/DeployStatus';
import { Time } from '@/components/Time';
import ClockIcon from '@/icons/ClockIcon';
import GitHubIcon from '@/icons/github.svg';
import VercelLogomark from '@/logos/vercel-logomark.svg';
import VercelWordmark from '@/logos/vercel-wordmark.svg';
import Button from '../Button';
import { FunctionDistribution } from './FunctionDistribution';
import { FunctionList } from './FunctionList';
import { getIntegrationName, isIntegration, type DeployMetadata } from './deployMetadata';

const integrationLogos = {
  vercel: {
    logomark: VercelLogomark,
    wordmark: VercelWordmark,
  },
} as const;

export type DeployCardProps = {
  id: string;
  appName: string;
  checksum: string;
  createdAt: string;
  deployedFunctions: { slug: string; name: string }[];
  removedFunctions: { slug: string; name: string }[];
  environmentSlug: string;
  error?: string | null | undefined;
  framework?: string | null;
  metadata: DeployMetadata;
  sdkLanguage: string;
  sdkVersion: string;
  status: string;
  url?: string | undefined | null;
};

export default function DeployCard({
  id,
  appName,
  createdAt,
  deployedFunctions,
  environmentSlug,
  framework,
  metadata,
  removedFunctions,
  sdkVersion,
  status,
  url,
}: DeployCardProps) {
  return (
    <div className="w-full px-8 py-8">
      <div className="rounded-lg bg-white shadow">
        <div className="flex w-full items-center justify-between border-b border-slate-100 px-4 py-4 pl-8">
          <div className="flex items-center gap-2">
            <DeployStatus status={status || ''} />
          </div>
          <span className="flex items-center gap-2 text-sm font-medium leading-none text-slate-600">
            <ClockIcon />
            <Time value={new Date(createdAt)} />
          </span>
        </div>
        <div className="px-6 py-4 pl-8">
          <div className="grid grid-cols-2 gap-4">
            {appName && <Labeled label="App Name">{appName}</Labeled>}
            {framework && <Labeled label="Framework">{capitalCase(framework)}</Labeled>}
            {sdkVersion && <Labeled label="SDK Version">{sdkVersion} </Labeled>}
            <Labeled label="Deploy ID">{id}</Labeled>

            {url && (
              <div className="col-span-2">
                <Labeled label="URL">{url}</Labeled>
              </div>
            )}
          </div>

          <FunctionDistribution
            activeCount={deployedFunctions.length}
            disabledCount={0}
            removedCount={removedFunctions.length}
          />
        </div>
      </div>

      {isIntegration(metadata) ? <IntegrationCard metadata={metadata} /> : null}

      <div className="mt-4 grid-cols-2 items-start gap-4 xl:grid">
        <FunctionList
          functions={deployedFunctions}
          baseHref={`/env/${environmentSlug}/functions`}
          status="active"
        />
        <FunctionList
          functions={removedFunctions}
          baseHref={`/env/${environmentSlug}/functions`}
          status="removed"
        />
      </div>
    </div>
  );
}

function IntegrationCard({ metadata }: { metadata: DeployMetadata }): JSX.Element {
  const integrationName = getIntegrationName(metadata);
  const IntegrationWordmark = integrationName ? integrationLogos[integrationName].wordmark : null;
  const IntegrationLogomark = integrationName ? integrationLogos[integrationName].logomark : null;

  // We need to cast the following URLs to Route because they are all external links.
  // See https://beta.nextjs.org/docs/configuring/typescript#statically-typed-links
  const projectUrl = metadata?.payload?.links?.project as Route;
  const deploymentUrl = metadata?.payload?.links?.deployment as Route;
  const url = metadata?.payload?.deployment?.url
    ? `https://${metadata.payload.deployment.url}`
    : '';

  // TODO - Support Gitlab and Bitbucket URLs
  const meta = metadata?.payload?.deployment?.meta;
  const repo = meta?.githubOrg && meta?.githubRepo ? `${meta?.githubOrg}/${meta?.githubRepo}` : '';
  // NOTE - This assumes it is public Github, not a privately hosted Github Enterprise
  const repoUrl = repo ? (`https://github.com/${repo}` as Route) : '';

  return (
    <div className="mt-4 rounded-lg bg-white shadow">
      <div className="flex w-full items-center justify-between border-b border-slate-100 py-4 pl-8 pr-4">
        <div className="flex items-center gap-2">
          {IntegrationWordmark ? (
            <IntegrationWordmark className="-ml-0.5 h-[19px] w-[84px]" />
          ) : (
            <span>{capitalCase(integrationName || '')} </span>
          )}
          <span className="rounded bg-sky-50 px-1.5 py-1 text-xs font-medium text-sky-600">
            Integration
          </span>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex gap-2 border-r border-slate-200/40 pr-2">
            {projectUrl && (
              <Button
                href={projectUrl}
                target="_blank"
                rel="noreferrer"
                variant="secondary"
                context="light"
              >
                {IntegrationLogomark ? (
                  <IntegrationLogomark className="-ml-0.5 h-4 w-4" />
                ) : (
                  <span>{capitalCase(integrationName || '')} </span>
                )}
                Project
              </Button>
            )}
            {deploymentUrl && (
              <Button
                // We need to cast this to Route because this is an external link.
                href={deploymentUrl}
                target="_blank"
                rel="noreferrer"
                variant="secondary"
                context="light"
              >
                {IntegrationLogomark ? (
                  <IntegrationLogomark className="-ml-0.5 h-4 w-4" />
                ) : (
                  <span>{capitalCase(integrationName || '')} </span>
                )}
                Deployment
              </Button>
            )}
          </div>
          {repoUrl && (
            <Button
              href={repoUrl}
              target="_blank"
              rel="noreferrer"
              variant="secondary"
              context="light"
            >
              <GitHubIcon className="-ml-0.5 h-4 w-4" />
              Repo
            </Button>
          )}
        </div>
      </div>
      <div className="flex flex-wrap gap-8 px-8 py-6">
        {url && (
          <Labeled label="URL">
            <a
              href={url}
              target="_blank"
              rel="noreferrer"
              className="text-sm font-medium text-slate-700 transition-all hover:text-indigo-500 hover:underline"
            >
              {url}
            </a>
          </Labeled>
        )}

        {repo && <Labeled label="Repository">{repo}</Labeled>}

        {meta?.githubCommitAuthorName && (
          <Labeled label="Commit Author">{meta.githubCommitAuthorName}</Labeled>
        )}
      </div>
    </div>
  );
}

function Labeled({ children, label }: { children: React.ReactNode; label: string }) {
  return (
    <label className="whitespace-nowrap text-xs font-semibold text-slate-400">
      {label}
      <div className="whitespace-nowrap text-sm font-medium text-slate-700">{children}</div>
    </label>
  );
}
