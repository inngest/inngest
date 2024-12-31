'use client';

import { useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button/index';
import { Card } from '@inngest/components/Card/Card';
import { Checkbox } from '@inngest/components/Checkbox/Checkbox';
import { Input } from '@inngest/components/Forms/Input';
import {
  RiArrowLeftSLine,
  RiArrowRightSLine,
  RiCloseLine,
  RiInformationLine,
} from '@remixicon/react';
import { useLocalStorage } from 'react-use';

import { OnboardingSteps } from '@/components/Onboarding/types';
import useOnboardingStep from '@/components/Onboarding/useOnboardingStep';
import { ONBOARDING_VERCEL_NEXT_URL } from '@/components/Onboarding/utils';
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
  const [custom, setCustom] = useState<string[]>([]);
  const { updateCompletedSteps } = useOnboardingStep();
  const [installingVercelFromOnboarding, setInstallingVercelFromOnboarding] = useLocalStorage(
    'installingVercelFromOnboarding',
    false
  );

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
    updateCompletedSteps(OnboardingSteps.DeployApp, {
      metadata: {
        completionSource: 'automatic',
        hostingProvider: 'vercel',
      },
    });

    router.push(
      `/integrations/vercel/callback/success?onSuccessRedirectURL=${
        installingVercelFromOnboarding ? ONBOARDING_VERCEL_NEXT_URL : searchParams.next
      }&source=${searchParams.source}` as Route
    );
    setInstallingVercelFromOnboarding(false);
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
              } border-subtle group px-6`}
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
                    onChange={(e) => setPath(p.id, e.target.value)}
                  />
                  <Button
                    size="small"
                    appearance="ghost"
                    kind="secondary"
                    icon={<RiCloseLine />}
                    className="absolute right-1 top-2 "
                    onClick={() => setCustom(custom.filter((c) => c !== p.id))}
                  />
                </div>
              ) : (
                <Button
                  className="hidden group-hover:block"
                  appearance="outlined"
                  label="Add custom path"
                  onClick={() => {
                    setCustom([...custom, p.id]);
                  }}
                />
              )}
            </div>
          ))}
          {projects.length > PAGE_SIZE && (
            <div className="row flex items-center justify-center p-2">
              <Button
                appearance="ghost"
                icon={
                  <RiArrowLeftSLine className="bg-canvasBase group-disabled:text-disabled text-basis h-6 w-6" />
                }
                disabled={page === 1}
                onClick={() => setPage(1)}
                className="group mr-1 h-6 w-6 p-0"
              />
              {[...Array(pages)].map((_, i) => (
                <Button
                  key={`page-${i}`}
                  appearance={page === i + 1 ? 'solid' : 'ghost'}
                  disabled={page === i + 1}
                  label={i + 1}
                  onClick={() => setPage(i + 1)}
                  className="text-basis disabled:bg-contrast disabled:text-onContrast mr-1 h-6 w-6 text-sm"
                />
              ))}
              <Button
                appearance="ghost"
                icon={
                  <RiArrowRightSLine className="bg-canvasBase group-disabled:text-disabled text-basis h-6 w-6" />
                }
                disabled={page === pages}
                onClick={() => setPage(pages)}
                className="h-6 w-6 p-0"
              />
            </div>
          )}
        </Card.Content>
      </Card>
      <div className="flex flex-row items-center justify-start rounded py-6">
        <RiInformationLine size={20} className="text-muted mr-1" />
        <div className="text-muted text-[15px] font-normal">
          More advanced configuration options will be available on Inngest dashboard after
          installation.
        </div>
      </div>
      <div>
        <Button
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
