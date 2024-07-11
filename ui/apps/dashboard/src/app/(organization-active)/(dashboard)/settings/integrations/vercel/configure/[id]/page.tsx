'use client';

import { useCallback, useEffect, useState } from 'react';
import NextLink from 'next/link';
import { useParams } from 'next/navigation';
import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button/Button';
import { NewButton } from '@inngest/components/Button/index';
import { Link } from '@inngest/components/Link/Link';
import { Switch, SwitchLabel, SwitchWrapper } from '@inngest/components/Switch/Switch';
import { RiAddLine, RiArrowRightSLine, RiDeleteBinLine, RiInformationLine } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import Input from '@/components/Forms/Input';
import {
  CreateVercelAppDocument,
  RemoveVercelAppDocument,
  UpdateVercelAppDocument,
} from '@/gql/graphql';
import LoadingIcon from '@/icons/LoadingIcon';
import { useProductionEnvironment } from '@/queries';
import { VercelDeploymentProtection, type VercelProject } from '../../VercelIntegration';
import { useVercelIntegration } from '../../useVercelIntegration';

const defaultPath = '/api/inngest';

export default function VercelConfigure() {
  const [{ data: env }] = useProductionEnvironment();
  const prodEnvID = env?.id;

  const { data, fetching, error: fetchError } = useVercelIntegration();
  const [, createVercelApp] = useMutation(CreateVercelAppDocument);
  const [, removeVercelApp] = useMutation(RemoveVercelAppDocument);
  const [, updateVercelApp] = useMutation(UpdateVercelAppDocument);
  const { id } = useParams<{ id: string }>();
  const [project, setProject] = useState<VercelProject & { updated?: boolean }>();

  // Represents the new, unsaved enablement value. For example, the value will
  // be "disabled" if the user disables the project but hasn't saved yet
  const [newEnablement, setNewEnablement] = useState<'disabled' | 'enabled'>();

  const [paths, setPaths] = useState([defaultPath]);

  //
  // For tracking loading states since urql does not offer that on mutations
  const [mutating, setMutating] = useState(false);

  useEffect(() => {
    if (!project) {
      const p = data.projects.find((p) => p.id === id);
      setProject(p);
      p?.servePath && setPaths(p.servePath.split(','));
    }
  }, [id, project, data.projects]);

  const joinedPaths = paths.join(',');
  const submit = useCallback(async () => {
    if (!prodEnvID) {
      console.error('no environment found');
      return;
    }
    if (!project) {
      console.error('no project found');
      return;
    }
    setMutating(true);

    let error: Error | undefined;

    // This is a quick-and-dirty (a.k.a. disgusting) solution to handle the way
    // we enable/disable Vercel projects. Enabling/disabling is actually done by
    // inserting/deleting a row in the DB
    if (newEnablement === 'enabled') {
      error = (
        await createVercelApp({
          input: {
            path: project.servePath,
            projectID: project.id,
            workspaceID: prodEnvID,
          },
        })
      ).error;
    } else if (newEnablement === 'disabled') {
      error = (
        await removeVercelApp({
          input: {
            projectID: project.id,
            workspaceID: prodEnvID,
          },
        })
      ).error;
    } else {
      error = (
        await updateVercelApp({
          input: {
            projectID: project.id,
            path: joinedPaths,
            protectionBypassSecret: project.protectionBypassSecret,
            originOverride: project.originOverride,
          },
        })
      ).error;
    }
    setMutating(false);

    if (error) {
      throw error;
    } else {
      setProject({ ...project, updated: false });
      setNewEnablement(undefined);
      //
      // TODO: new designs
      toast.success('Changes saved!');
    }
  }, [
    createVercelApp,
    joinedPaths,
    newEnablement,
    prodEnvID,
    project,
    removeVercelApp,
    updateVercelApp,
  ]);

  // Only show the "extra" settings (stuff besides the enablement toggle) if the
  // project is enabled. This is mostly because not all extra inputs are saved
  // when enabling a project (e.g. the protection bypass secret)
  const areExtraSettingsVisible = project?.isEnabled && newEnablement !== 'enabled';

  if (fetching) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  if (fetchError) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <Alert severity="error">{fetchError.message}</Alert>
      </div>
    );
  }

  if (!data && !project) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <Alert severity="error">Vercel project not found!</Alert>
      </div>
    );
  }

  return (
    <div className="mx-auto mt-6 flex w-[800px] flex-col p-8">
      {project && (
        <div className="flex flex-col">
          <div className="flex flex-row items-center justify-start">
            <NextLink href="/settings/integrations">
              <div className="text-subtle text-base">All integrations</div>
            </NextLink>
            <RiArrowRightSLine className="text-disabled h-4" />
            <NextLink href="/settings/integrations/vercel">
              <div className="text-subtle text-base">Vercel</div>
            </NextLink>
            <RiArrowRightSLine className="text-disabled h-4" />
            <div className="text-basis text-base">{project.name}</div>
          </div>
          <div className="text-basis mb-2 mt-6 text-2xl font-medium">{project.name}</div>
          {project.ssoProtection?.deploymentType ===
            VercelDeploymentProtection.ProdDeploymentURLsAndAllPreviews && (
            <div className="text-accent-intense mb-7 flex flex-row items-center justify-start text-sm leading-tight">
              <RiInformationLine className="mr-1 h-4 w-4" />
              Vercel Deployment Protection might block syncing. Use the deployment protection key
              option below to bypass.
            </div>
          )}

          <div className="border-subtle flex flex-col gap-2 rounded-lg border p-6">
            <div className="text-basis text-lg font-medium">Project status</div>
            <div className="text-subtle text-base font-normal">
              This determines whether or not Inngest will communicate with your Vercel application.
            </div>
            <SwitchWrapper>
              <Switch
                checked={project.isEnabled}
                className="data-[state=checked]:bg-primary-moderate"
                onCheckedChange={(checked) => {
                  setProject({ ...project, isEnabled: checked, updated: true });
                  setNewEnablement((prev) => {
                    if (!prev) {
                      // Only change
                      return checked ? 'enabled' : 'disabled';
                    }
                    return undefined;
                  });
                }}
              />
              <SwitchLabel htmlFor="override" className="text-subtle text-sm leading-tight">
                {project.isEnabled ? 'Enabled' : 'Disabled'}
              </SwitchLabel>
            </SwitchWrapper>
          </div>

          {areExtraSettingsVisible && (
            <>
              <div className="border-subtle mt-4 flex flex-col gap-2 rounded-lg border p-6">
                <div className="text-basis text-lg font-medium">Path information</div>
                <div className="text-subtle text-base font-normal">
                  Each Vercel project can serve one or more Inngest apps available on different URL
                  paths.
                </div>
                {paths.map((path, i) => (
                  <div key={`serve-path-${i}`} className="flex flex-row items-center justify-start">
                    <div className="mr-2 w-full">
                      <Input
                        defaultValue={path}
                        className="text-basis h-10 w-full px-2 py-2 text-base"
                        onChange={({ target: { value } }) => {
                          setPaths(paths.map((p, n) => (i === n ? value : p)));
                          setProject({ ...project, updated: true });
                        }}
                      />
                    </div>

                    {paths.length > 1 && (
                      <Button
                        kind="danger"
                        appearance="outlined"
                        icon={<RiDeleteBinLine className="h-5 w-5" />}
                        className="h-10 w-10"
                        onClick={() => setPaths(paths.filter((_, n) => n !== i))}
                      />
                    )}
                  </div>
                ))}
                <div>
                  <NewButton
                    appearance="outlined"
                    icon={<RiAddLine className="mr-1" />}
                    iconSide="left"
                    label="Add new path"
                    className="mt-3"
                    onClick={() => {
                      setProject({ ...project, updated: true });
                      setPaths([...paths, '']);
                    }}
                  />
                </div>
              </div>
              <div className="flex flex-row gap-4">
                <div
                  className={`border-subtle mt-4 flex w-full flex-col gap-2 rounded-lg border p-6 ${
                    project.ssoProtection?.deploymentType !==
                      VercelDeploymentProtection.ProdDeploymentURLsAndAllPreviews && 'bg-disabled'
                  }`}
                >
                  <div className="text-basis text-lg font-medium">Deployment protection key</div>
                  <div className="text-subtle text-base font-normal">
                    Used to bypass deployment protection.{' '}
                    <Link href="https://www.inngest.com/docs/deploy/vercel#bypassing-deployment-protection">
                      Learn more
                    </Link>
                  </div>
                  <Input
                    className={`text-basis mt-4 h-10 px-2 py-2 text-base ${
                      project.ssoProtection?.deploymentType !==
                        VercelDeploymentProtection.ProdDeploymentURLsAndAllPreviews &&
                      'border-subtle bg-disabled'
                    }`}
                    readonly={
                      project.ssoProtection?.deploymentType !==
                      VercelDeploymentProtection.ProdDeploymentURLsAndAllPreviews
                    }
                    onChange={({ target: { value } }) =>
                      setProject({ ...project, protectionBypassSecret: value, updated: true })
                    }
                    value={project.protectionBypassSecret ?? ''}
                  />
                </div>
                <div className="border-subtle mt-4 flex w-full flex-col gap-2 rounded-lg border p-6">
                  <div className="text-basis text-lg font-medium">
                    Custom Production Domain <span className="text-sublte text-xs">(optional)</span>
                  </div>
                  <div className="text-subtle text-base font-normal">
                    Set a custom domain to use for production instead of the URL generated by
                    Vercel.
                  </div>
                  <Input
                    className="text-basis mt-4 h-10 px-2 py-2 text-base"
                    placeholder="Add custom domain"
                    value={project.originOverride ?? ''}
                    onChange={({ target: { value } }) =>
                      setProject({ ...project, originOverride: value, updated: true })
                    }
                  />
                </div>
              </div>
            </>
          )}

          <div className="mt-6 flex flex-row items-center justify-start">
            <NewButton
              label="Save configuration"
              disabled={!project.updated}
              onClick={submit}
              loading={mutating}
            />
            {project.updated && (
              <div className="text-subtle ml-4 text-[13px] leading-tight">Unsaved changes</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
