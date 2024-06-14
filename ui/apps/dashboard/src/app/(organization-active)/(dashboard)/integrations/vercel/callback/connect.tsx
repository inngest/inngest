'use client';

import { useState, type ChangeEvent } from 'react';
import { Button, NewButton } from '@inngest/components/Button/index';
import { Card } from '@inngest/components/Card/Card';
import { cn } from '@inngest/components/utils/classNames';
import { RiArrowLeftSLine, RiArrowRightSLine, RiInformationLine } from '@remixicon/react';

import type VercelIntegration from '../../../settings/integrations/vercel/VercelIntegration';
import useUpdateVercelIntegration from '../../../settings/integrations/vercel/useUpdateVercelIntegration';
import type { VercelCallbackProps } from './page';

const PAGE_SIZE = 4;

export default function Connect({
  searchParams,
  integrations,
}: VercelCallbackProps & { integrations: VercelIntegration }) {
  const [projects, setProjects] = useState(integrations.projects);
  const [saving, setSaving] = useState(false);
  const [page, setPage] = useState(1);
  const start = (page - 1) * PAGE_SIZE;

  const updateVercelIntegration = useUpdateVercelIntegration(integrations);

  const check = ({ target: { id, checked } }: ChangeEvent<HTMLInputElement>) => {
    setProjects(projects.map((p) => ({ ...p, isEnabled: p.id === id ? !!checked : p.isEnabled })));
  };

  // const dummy = [...Array(20)].map((_, i) =>
  //   i === 0
  //     ? projects[0]
  //     : { ...projects[0], id: `id-${i}`, name: `project-${i}`, isEnabled: false }
  // );
  const pages = Math.ceil(projects?.length / PAGE_SIZE);
  const end = Math.min(start + PAGE_SIZE, projects?.length);

  const submit = async () => {
    await updateVercelIntegration({
      ...integrations,
      projects,
    });
    setSaving(false);
  };

  return (
    <>
      <Card className="w-full border-slate-200">
        <Card.Header className="bg-slate-100">
          <div className="flex flex-row items-center justify-between">
            <div className="text-base text-gray-900">Project list</div>
            <div className="text-sm font-medium text-slate-400">
              {projects.length || 0} projects selected
            </div>
          </div>
        </Card.Header>

        <Card.Content className="p-0">
          {[...projects?.slice(start, end)]?.map((p: any, i) => (
            <div
              className={`flow-row flex h-[72px] items-center justify-start ${
                i !== end && 'border-b'
              } border-slate-200 px-6`}
            >
              <input
                id={p!.id}
                type="checkbox"
                className="mr-2 h-4 w-4 text-indigo-600 focus:outline-0 focus:ring-0"
                onChange={check}
                checked={p.isEnabled}
              />

              <div className="text-base font-normal leading-7 text-slate-700">{p.name}</div>
            </div>
          ))}
          {projects?.length > PAGE_SIZE && (
            <div className="row flex flex items-center justify-center p-2">
              <NewButton
                appearance="ghost"
                icon={<RiArrowLeftSLine className="mr h-7 w-7" />}
                disabled={page === 1}
                onClick={() => setPage(1)}
              />
              {[...Array(pages)].map((_, i) => (
                <NewButton
                  key={`page-${i}`}
                  appearance={page === i + 1 ? 'solid' : 'ghost'}
                  disabled={page === i + 1}
                  label={i + 1}
                  onClick={() => setPage(i + 1)}
                />
              ))}
              <NewButton
                appearance="ghost"
                icon={<RiArrowRightSLine className="h-7 w-7" />}
                disabled={page === pages}
                onClick={() => setPage(pages)}
              />
            </div>
          )}
        </Card.Content>
      </Card>
      <div className="flex flex-row items-center justify-start rounded py-6">
        <RiInformationLine size={20} className="mr-2 text-slate-500" />
        <div className="text-[15px] font-normal text-slate-500">
          More advanced configuration options will be available on Inngest dashboard after
          installation.
        </div>
      </div>
      <div>
        <NewButton
          kind="primary"
          appearance="solid"
          size="medium"
          label="Save configuration"
          loading={saving}
          onClick={() => {
            setSaving(true);
            submit();
          }}
        />
      </div>
    </>
  );
}
