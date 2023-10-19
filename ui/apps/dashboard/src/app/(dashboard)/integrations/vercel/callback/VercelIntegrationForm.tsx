'use client';

import { useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { Switch } from '@headlessui/react';
import { Button } from '@inngest/components/Button';

import type VercelIntegration from '@/app/(dashboard)/settings/integrations/vercel/VercelIntegration';
import useUpdateVercelIntegration from '@/app/(dashboard)/settings/integrations/vercel/useUpdateVercelIntegration';
import Input from '@/components/Forms/Input';
import cn from '@/utils/cn';

type VercelIntegrationFormProps = {
  vercelIntegration: VercelIntegration;
  onSuccessRedirectURL: string;
};

export default function VercelIntegrationForm({
  vercelIntegration,
  onSuccessRedirectURL,
}: VercelIntegrationFormProps) {
  const router = useRouter();
  const [currentPage, setCurrentPage] = useState(1);
  const updateVercelIntegration = useUpdateVercelIntegration(vercelIntegration);

  const projects = vercelIntegration.projects;
  const maxProjectsPerPage = 4;
  const totalPages = Math.ceil(projects.length / maxProjectsPerPage);
  const projectsOnPage = projects.slice(
    (currentPage - 1) * maxProjectsPerPage,
    currentPage * maxProjectsPerPage
  );
  const hasPages = totalPages > 1;
  const hasPreviousPage = currentPage > 1;
  const hasNextPage = currentPage < totalPages;

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

    await updateVercelIntegration({
      ...vercelIntegration,
      projects: updatedProjects,
    });
    router.push(
      `/integrations/vercel/callback/success?onSuccessRedirectURL=${onSuccessRedirectURL}` as Route
    );
  }

  return (
    <form method="post" onSubmit={handleSubmit} className="-mx-2 space-y-6">
      <div className="space-y-4">
        <ul className="flex flex-col gap-3 rounded-md bg-slate-100 p-2">
          {projects.map((project) => {
            const isVisible = projectsOnPage.includes(project);
            return (
              <li
                className={cn('flex items-center justify-between', !isVisible && 'hidden')}
                key={project.id}
              >
                <ProjectEnableToggle
                  projectID={project.id}
                  initialIsEnabled={true}
                  projectName={project.name}
                />
                <ProjectServePathInput projectID={project.id} servePath={project.servePath} />
              </li>
            );
          })}
        </ul>
        {hasPages && (
          <nav className="space-x-2 text-right" aria-label="Pagination">
            <Button
              btnAction={() => setCurrentPage(currentPage - 1)}
              appearance="outlined"
              disabled={!hasPreviousPage}
              label="Prev"
            />
            <span className="isolate inline-flex rounded-md shadow-sm">
              {Array.from({ length: totalPages }, (_, index) => {
                const isCurrentPage = index === currentPage - 1;
                const isFirstPage = index === 0;
                const isLastPage = index === totalPages - 1;
                return (
                  <Button
                    btnAction={() => setCurrentPage(index + 1)}
                    key={index}
                    appearance="outlined"
                    aria-current={isCurrentPage ? 'page' : undefined}
                    className={cn(
                      'relative z-10 rounded-none rounded-l-md focus:z-20 focus:outline-offset-0',
                      isFirstPage && 'rounded-l-md',
                      isLastPage && 'rounded-r-md',
                      isCurrentPage
                        ? 'hover:bg-white focus-visible:outline  focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500'
                        : 'bg-slate-100'
                    )}
                    label={index + 1}
                  />
                );
              })}
            </span>
            <Button
              appearance="outlined"
              disabled={!hasNextPage}
              btnAction={() => setCurrentPage(currentPage + 1)}
              label="Next"
            />
          </nav>
        )}
      </div>
      <div className="flex items-center justify-between">
        <p className="max-w-md text-sm">
          Once saved, Inngest will automatically configure your functions to be run whenever you
          deploy to Vercel.{' '}
          <a
            className="underline"
            target="_blank"
            href="https://www.inngest.com/docs/deploy/vercel"
          >
            Find out more
          </a>{' '}
          about the Vercel integration.
        </p>
        <Button type="submit" label="Save Configuration" />
      </div>
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
    <Switch.Group as="div" className="flex items-center gap-4">
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
      <Switch.Label as="span" className="text-sm font-medium text-slate-800" passive>
        {projectName}
      </Switch.Label>
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
