'use client';

import { useCallback, useEffect, useState } from 'react';
import NextLink from 'next/link';
import { useParams } from 'next/navigation';
import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button/index';
import { Input } from '@inngest/components/Forms/Input';
import { Link } from '@inngest/components/Link/Link';
import { Switch, SwitchLabel, SwitchWrapper } from '@inngest/components/Switch/Switch';
import { RiAddLine, RiArrowRightSLine, RiDeleteBinLine, RiInformationLine } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import {
  CreateVercelAppDocument,
  RemoveVercelAppDocument,
  UpdateVercelAppDocument,
  VercelDeploymentProtection,
  type VercelProject,
} from '@/gql/graphql';
import LoadingIcon from '@/icons/LoadingIcon';
import { useDefaultEnvironment } from '@/queries';
import { useVercelIntegration } from '../../useVercelIntegration';

const defaultPath = '/api/inngest';

export default function VercelConfigure() {
  const [{ data: defaultEnv }] = useDefaultEnvironment();
  const defaultEnvID = defaultEnv?.id;

  const { data, isLoading, error: fetchError } = useVercelIntegration();
  const [, createVercelApp] = useMutation(CreateVercelAppDocument);
  const [, removeVercelApp] = useMutation(RemoveVercelAppDocument);
  const [, updateVercelApp] = useMutation(UpdateVercelAppDocument);
  const { id } = useParams<{ id: string }>();
  const [originalProject, setOriginalProject] = useState<VercelProject>();
  const [project, setProject] = useState<VercelProject>();
  const [updated, setUpdated] = useState(false);
  const [notFound, setNotFound] = useState(false);
  const [paths, setPaths] = useState([defaultPath]);

  //
  // For tracking loading states since urql does not offer that on mutations
  const [mutating, setMutating] = useState(false);

  useEffect(() => {
    if (originalProject || !data) {
      return;
    }

    //
    // Whenever we load or update a project, store the original state.
    // We need to do this so we only issue add/remove when those changes
    // have been made as those operations are not idempotent upstream.
    const p = data.projects.find((p) => p.projectID === id);
    if (p) {
      p.servePath && setPaths(p.servePath.split(','));
      setOriginalProject(p);
      setNotFound(false);
    } else {
      setNotFound(true);
    }
  }, [id, data, originalProject]);

  useEffect(() => {
    originalProject && setProject({ ...originalProject });
  }, [originalProject]);

  useEffect(() => {
    setUpdated(JSON.stringify(originalProject) !== JSON.stringify(project));
    //
    // we only want to track updates on project changes, not originalProject
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project]);

  useEffect(() => {
    project && setProject({ ...project, servePath: paths.join(',') });
    //
    // we only want to track updates on paths changes, not project
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paths]);

  const submit = useCallback(async () => {
    if (!defaultEnvID) {
      console.error('no environment found');
      return;
    }
    if (!project) {
      console.error('no project found');
      return;
    }
    setMutating(true);

    //
    // if there are other change and isEnabled is still false, this is a no-op
    if (project.isEnabled === originalProject?.isEnabled && !project.isEnabled) {
      setMutating(false);
      return;
    }

    //
    // enable/disable are actually insert/delete in the db so we need to detect
    // and call those specifically
    const { error } =
      project.isEnabled !== originalProject?.isEnabled && project.isEnabled
        ? await createVercelApp({
            input: {
              path: project.servePath,
              projectID: project.projectID,
              workspaceID: defaultEnvID,
            },
          })
        : project.isEnabled !== originalProject?.isEnabled && !project.isEnabled
        ? await removeVercelApp({
            input: {
              projectID: project.projectID,
              workspaceID: defaultEnvID,
            },
          })
        : await updateVercelApp({
            input: {
              projectID: project.projectID,
              path: project.servePath,
              protectionBypassSecret: project.protectionBypassSecret,
              originOverride: project.originOverride ? project.originOverride : undefined,
            },
          });

    setMutating(false);

    if (error) {
      throw error;
    } else {
      setOriginalProject(project);

      //
      // TODO: new designs
      toast.success('Changes saved!');
    }
  }, [createVercelApp, defaultEnvID, project, removeVercelApp, updateVercelApp, originalProject]);

  // Only show the "extra" settings (stuff besides the enablement toggle) if the
  // project is enabled. This is mostly because not all extra inputs are saved
  // when enabling a project (e.g. the protection bypass secret)
  const areExtraSettingsVisible = project?.isEnabled;

  if (isLoading) {
    return (
      <div className="mt-6 flex h-full w-full items-center justify-center">
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

  if (notFound) {
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
              <div className="text-muted text-base">All integrations</div>
            </NextLink>
            <RiArrowRightSLine className="text-disabled h-4" />
            <NextLink
              href={{
                pathname: '/settings/integrations/vercel',
                query: { ts: Date.now() },
              }}
              prefetch={false}
            >
              <div className="text-muted text-base">Vercel</div>
            </NextLink>
            <RiArrowRightSLine className="text-disabled h-4" />
            <div className="text-basis text-base">{project.name}</div>
          </div>
          <div className="text-basis mb-2 mt-6 text-2xl font-medium">{project.name}</div>
          {project.deploymentProtection !== VercelDeploymentProtection.Disabled && (
            <div className="text-accent-intense mb-7 flex flex-row items-center justify-start text-sm leading-tight">
              <RiInformationLine className="mr-1 h-4 w-4" />
              Vercel Deployment Protection might block syncing. Use the deployment protection key
              option below to bypass.
            </div>
          )}

          {project.canChangeEnabled && (
            <div className="border-subtle flex flex-col gap-2 rounded-md border p-6">
              <div className="text-basis text-lg font-medium">Project status</div>
              <div className="text-muted text-base font-normal">
                This determines whether or not Inngest will communicate with your Vercel
                application.
              </div>
              <SwitchWrapper>
                <Switch
                  checked={project.isEnabled}
                  className="data-[state=checked]:bg-primary-moderate cursor-pointer"
                  onCheckedChange={(checked) => {
                    setProject({ ...project, isEnabled: checked });
                  }}
                />
                <SwitchLabel htmlFor="override" className="text-muted text-sm leading-tight">
                  {project.isEnabled ? 'Enabled' : 'Disabled'}
                </SwitchLabel>
              </SwitchWrapper>
            </div>
          )}

          {areExtraSettingsVisible && (
            <>
              <div className="border-subtle mt-4 flex flex-col gap-2 rounded-md border p-6">
                <div className="text-basis text-lg font-medium">Path information</div>
                <div className="text-muted text-base font-normal">
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
                          setProject(project);
                        }}
                      />
                    </div>

                    {paths.length > 1 && (
                      <Button
                        kind="danger"
                        appearance="outlined"
                        icon={<RiDeleteBinLine className="h-5 w-5" />}
                        iconSide="left"
                        className="h-10 w-10"
                        onClick={() => setPaths(paths.filter((_, n) => n !== i))}
                      />
                    )}
                  </div>
                ))}
                <div>
                  <Button
                    appearance="outlined"
                    icon={<RiAddLine className="mr-1" />}
                    iconSide="left"
                    label="Add new path"
                    className="mt-3"
                    onClick={() => {
                      setProject({ ...project });
                      setPaths([...paths, '']);
                    }}
                  />
                </div>
              </div>
              <div className="flex flex-row gap-4">
                <div
                  className={`border-subtle mt-4 flex w-full flex-col gap-2 rounded-md border p-6 ${
                    project.deploymentProtection === VercelDeploymentProtection.Disabled &&
                    'bg-disabled'
                  }`}
                >
                  <div className="text-basis text-lg font-medium">Deployment protection key</div>
                  <div className="text-muted text-base font-normal">
                    Used to bypass deployment protection.{' '}
                    <Link
                      size="medium"
                      target="_blank"
                      href="https://www.inngest.com/docs/deploy/vercel#bypassing-deployment-protection"
                    >
                      Learn more
                    </Link>
                  </div>
                  <Input
                    className={`text-basis mt-4 h-10 px-2 py-2 text-base ${
                      project.deploymentProtection === VercelDeploymentProtection.Disabled &&
                      'border-subtle bg-disabled'
                    }`}
                    readOnly={project.deploymentProtection === VercelDeploymentProtection.Disabled}
                    onChange={({ target: { value } }) =>
                      setProject({ ...project, protectionBypassSecret: value })
                    }
                    value={project.protectionBypassSecret ?? ''}
                  />
                </div>
                <div className="border-subtle mt-4 flex w-full flex-col gap-2 rounded-md border p-6">
                  <div className="text-basis text-lg font-medium">
                    Custom Production Domain <span className="text-sublte text-xs">(optional)</span>
                  </div>
                  <div className="text-muted text-base font-normal">
                    Set a custom domain to use for production instead of the URL generated by
                    Vercel.
                  </div>
                  <Input
                    className="text-basis mt-4 h-10 px-2 py-2 text-base"
                    placeholder="Add custom domain"
                    value={project.originOverride ?? ''}
                    onChange={({ target: { value } }) =>
                      setProject({ ...project, originOverride: value })
                    }
                  />
                </div>
              </div>
            </>
          )}

          <div className="mt-6 flex flex-row items-center justify-start">
            <Button
              label="Save configuration"
              disabled={!updated}
              onClick={submit}
              loading={mutating}
            />
            {updated && (
              <div className="text-muted ml-4 text-[13px] leading-tight">Unsaved changes</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
