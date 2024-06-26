'use client';

import { useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { NewButton } from '@inngest/components/Button/index';
import { Card } from '@inngest/components/Card/Card';
import { Checkbox } from '@inngest/components/Checkbox/Checkbox';
import {
  RiArrowLeftSLine,
  RiArrowRightSLine,
  RiCloseLine,
  RiInformationLine,
} from '@remixicon/react';

import Input from '@/components/Forms/Input';
import type VercelIntegration from '../../../settings/integrations/vercel/VercelIntegration';
import useUpdateVercelIntegration from '../../../settings/integrations/vercel/useUpdateVercelIntegration';
import type { VercelCallbackProps } from './page';

const PAGE_SIZE = 4;

export default function Connect({
  searchParams,
  integrations,
}: VercelCallbackProps & { integrations: VercelIntegration }) {
  const router = useRouter();
  const [projects, setProjects] = useState(integrations.projects);
  const [saving, setSaving] = useState(false);
  const [page, setPage] = useState(1);
  const start = (page - 1) * PAGE_SIZE;
  const [hover, setHover] = useState(null);
  const [custom, setCustom] = useState<string[]>([]);

  const updateVercelIntegration = useUpdateVercelIntegration(integrations);

  const check = (id: string, enabled: boolean) =>
    setProjects(projects.map((p) => ({ ...p, isEnabled: p.id === id ? enabled : p.isEnabled })));

  const setPath = (id: string, value: string) =>
    setProjects(projects.map((p) => ({ ...p, servePath: p.id === id ? value : p.servePath })));

  const pages = Math.ceil(projects.length / PAGE_SIZE);
  const end = Math.min(start + PAGE_SIZE, projects.length);

  const submit = async () => {
    await updateVercelIntegration({
      ...integrations,
      projects,
    });
    setSaving(false);
    router.push(
      `/integrations/vercel/callback/success?onSuccessRedirectURL=${searchParams.next}` as Route
    );
  };

  return (
    <>
      <Card className="w-full">
        <Card.Header className="bg-canvasSubtle">
          <div className="flex flex-row items-center justify-between">
            <div className="text-basis text-base">Project list</div>
            <div className="text-disabled text-sm font-medium">
              {projects.filter((p) => p.isEnabled).length || 0} projects selected
            </div>
          </div>
        </Card.Header>

        <Card.Content className="p-0">
          {[...projects.slice(start, end)].map((p: any, i) => (
            <div
              key={`project-list-${i}`}
              className={`flex h-[72px] flex-row items-center justify-between ${
                i !== end && 'border-b'
              } border-subtle px-6`}
              onMouseOver={() => setHover(p.id)}
              onMouseLeave={() => setHover(null)}
            >
              <div className="flex flex-row items-center justify-start">
                <Checkbox
                  id={p.id}
                  className="mr-2 h-4 w-4"
                  onCheckedChange={() => check(p.id, !p.isEnabled)}
                  checked={p.isEnabled}
                />
                <div className="text-basis text-base font-normal">{p.name}</div>
              </div>
              {custom.includes(p.id) ? (
                <div className="relative">
                  <Input
                    required={true}
                    placeholder="Add custom path"
                    className="h-10 w-96"
                    showError={false}
                    onChange={(e) => setPath(p.id, e.target.value)}
                  />
                  <NewButton
                    size="small"
                    appearance="ghost"
                    kind="secondary"
                    icon={<RiCloseLine />}
                    className="absolute right-1 top-2"
                    onClick={() => setCustom(custom.filter((c) => c !== p.id))}
                  />
                </div>
              ) : (
                hover === p.id && (
                  <NewButton
                    appearance="outlined"
                    label="Add custom path"
                    onClick={() => {
                      setCustom([...custom, p.id]);
                    }}
                  />
                )
              )}
            </div>
          ))}
          {projects.length > 0 && (
            <div className="row flex items-center justify-center p-2">
              <NewButton
                appearance="ghost"
                icon={<RiArrowLeftSLine className="disabled:text-disabled text-basis" />}
                disabled={page === 1}
                onClick={() => setPage(1)}
                className="mr-1 h-6"
              />
              {[...Array(pages)].map((_, i) => (
                <NewButton
                  key={`page-${i}`}
                  appearance={page === i + 1 ? 'solid' : 'ghost'}
                  disabled={page === i + 1}
                  label={i + 1}
                  onClick={() => setPage(i + 1)}
                  className="text-basis bg-contrast mr-1 h-6 text-sm disabled:bg-black disabled:text-white"
                />
              ))}
              <NewButton
                appearance="ghost"
                icon={<RiArrowRightSLine className="disabled:text-disabled text-basis " />}
                disabled={page === pages}
                onClick={() => setPage(pages)}
                className="h-6"
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
