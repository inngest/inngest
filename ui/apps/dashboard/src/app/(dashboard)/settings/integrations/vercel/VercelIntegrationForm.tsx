'use client';

import { useState } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { Switch } from '@headlessui/react';
import { ArrowPathIcon, ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';

import {
  VercelDeploymentProtection,
  type VercelIntegration,
} from '@/app/(dashboard)/settings/integrations/vercel/VercelIntegration';
import AppLink from '@/components/AppLink';
import Input from '@/components/Forms/Input';
import VercelLogomark from '@/logos/vercel-logomark.svg';
import VercelWordmark from '@/logos/vercel-wordmark.svg';
import cn from '@/utils/cn';
import useUpdateVercelIntegration from './useUpdateVercelIntegration';

type VercelIntegrationFormProps = {
  vercelIntegration: VercelIntegration;
};

export default function VercelIntegrationForm({ vercelIntegration }: VercelIntegrationFormProps) {
  const router = useRouter();
  const updateVercelIntegration = useUpdateVercelIntegration(vercelIntegration);
  const projects = vercelIntegration.projects;

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const form = event.currentTarget;
    const formData = new FormData(form);

    const updatedProjects = projects.map((project) => {
      const servePath = (formData.get(`${project.id}.servePath`) as string | null) || undefined;
      const isEnabled = formData.get(`${project.id}.isEnabled`) === 'on';

      return {
        ...project,
        servePath,
        isEnabled,
      };
    });

    const updateVercelIntegrationPromise = updateVercelIntegration({
      ...vercelIntegration,
      projects: updatedProjects,
    });
    toast.promise(updateVercelIntegrationPromise, {
      loading: 'Loading...',
      success: () => {
        router.refresh();
        return 'Configuration saved!';
      },
      error: 'Could not save configuration. Please try again later.',
    });
  }

  if (!vercelIntegration.enabled) {
    return <AddIntegrationPage />;
  }

  return (
    <form method="post" onSubmit={handleSubmit} className="space-y-12 p-12">
      <header className="space-y-5 lg:flex lg:justify-between lg:gap-6 lg:space-y-0">
        <div className="space-y-4">
          <VercelWordmark className="h-6" />
          <p className="max-w-xl text-sm text-slate-700">
            Click “Enable” for each project that you have Ingest functions. You can optionally
            specify a custom serve route (see docs) other than the default.
          </p>
        </div>
        <div className="flex shrink-0 gap-2.5">
          <div className="flex gap-2">
            <Button
              appearance="outlined"
              btnAction={() => router.refresh()}
              icon={<ArrowPathIcon className=" text-slate-500" />}
              label="Refresh Project List"
            />
            <Button
              appearance="outlined"
              href={'https://vercel.com/integrations/inngest' as Route}
              icon={<VercelLogomark />}
              label="Go to Vercel"
            />
          </div>
          <div className="h-8 w-px bg-slate-200" aria-hidden="true" />
          <Button type="submit" label="Save Configuration" />
        </div>
      </header>
      <main>
        <ul className="flex flex-wrap gap-10">
          {projects.map((project) => (
            <li className="flex flex-col gap-4" key={project.id}>
              <ProjectEnableToggle
                projectID={project.id}
                initialIsEnabled={project.isEnabled}
                projectName={project.name}
              />
              <ProjectServePathInput projectID={project.id} servePath={project.servePath} />
              {(project.ssoProtection?.deploymentType ===
                VercelDeploymentProtection.ProdDeploymentURLsAndAllPreviews ||
                project.ssoProtection?.deploymentType === VercelDeploymentProtection.Previews) && (
                <div className="flex items-center gap-2 text-sm">
                  <ExclamationCircleIcon className="h-4 text-amber-500" /> Deployment protection
                  enabled -{' '}
                  <AppLink
                    href="https://www.inngest.com/docs/deploy/vercel#bypassing-deployment-protection"
                    target="_blank"
                  >
                    Learn more
                  </AppLink>
                </div>
              )}
            </li>
          ))}
        </ul>
      </main>
    </form>
  );
}

type ProjectEnableToggleProps = {
  projectID: string;
  projectName: string;
  initialIsEnabled: boolean;
};

function ProjectEnableToggle({
  projectID,
  projectName,
  initialIsEnabled,
}: ProjectEnableToggleProps) {
  return (
    <Switch.Group as="div" className="flex items-center justify-between">
      <Switch.Label as="span" className="font-medium text-slate-800" passive>
        {projectName}
      </Switch.Label>
      <Switch
        name={`${projectID}.isEnabled`}
        defaultChecked={initialIsEnabled}
        className="relative inline-flex h-5 w-10 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent bg-white ring-1 ring-slate-200 transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
      >
        {({ checked }) => (
          <span
            className={cn(
              checked ? 'translate-x-5 bg-indigo-500' : 'translate-x-0 bg-slate-500',
              'pointer-events-none relative inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out'
            )}
          >
            <span
              className={cn(
                checked ? 'opacity-0 duration-100 ease-out' : 'opacity-100 duration-200 ease-in',
                'absolute inset-0 flex h-full w-full items-center justify-center transition-opacity'
              )}
              aria-hidden="true"
            >
              <span className="h-0.5 w-2.5 rounded-full bg-white" />
            </span>
            <span
              className={cn(
                checked ? 'opacity-100 duration-200 ease-in' : 'opacity-0 duration-100 ease-out',
                'absolute inset-0 flex h-full w-full items-center justify-center transition-opacity'
              )}
              aria-hidden="true"
            >
              <span className="h-1.5 w-1.5 rounded-full bg-white" />
            </span>
          </span>
        )}
      </Switch>
    </Switch.Group>
  );
}

type ProjectServePathInputProps = {
  projectID: string;
  servePath?: string;
};

function ProjectServePathInput({ projectID, servePath }: ProjectServePathInputProps) {
  const [isEmpty, setIsEmpty] = useState(!servePath);

  return (
    <div className="relative">
      <Input
        className={cn('w-80', !servePath && 'pr-20')}
        size="lg"
        name={`${projectID}.servePath`}
        placeholder="/api/inngest"
        defaultValue={servePath}
        onChange={(event) => setIsEmpty(event.target.value === '')}
      />
      {isEmpty && (
        <div className="absolute inset-y-2 right-2 inline-flex items-center rounded bg-indigo-500 px-2 text-xs text-white">
          Default
        </div>
      )}
    </div>
  );
}

function AddIntegrationPage() {
  return (
    <div className="space-y-12 p-12">
      <header className="space-y-5 lg:flex lg:justify-between lg:gap-6 lg:space-y-0">
        <div className="space-y-4">
          <VercelWordmark className="h-6" />
          <p className="max-w-xl text-sm text-slate-700">
            Inngest enables you to host your functions on Vercel using their serverless functions
            platform. This allows you to deploy your Inngest functions right alongside your existing
            website and API functions running on Vercel.
          </p>
          <p className="max-w-xl text-sm text-slate-700">
            Inngest will call your functions securely via HTTP request on-demand, whether triggered
            by an event or on a schedule in the case of cron jobs.
          </p>
        </div>
      </header>
      <main>
        <div className="flex gap-2">
          <Button
            kind="primary"
            href={'https://vercel.com/integrations/inngest' as Route}
            target="_blank"
            icon={<VercelLogomark />}
            label="Install Vercel Integration"
          />
          <Button
            appearance="outlined"
            href={'https://www.inngest.com/docs/deploy/vercel?ref=app-integrations' as Route}
            target="_blank"
            label="Read the Docs"
          />
        </div>
      </main>
    </div>
  );
}
