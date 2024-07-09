'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Badge } from '@inngest/components/Badge/Badge';
import { NewButton } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';
import { Link } from '@inngest/components/Link/Link';
import { Select } from '@inngest/components/Select/Select';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiInformationLine, RiRefreshLine } from '@remixicon/react';

import { VercelDeploymentProtection } from './VercelIntegration';
import { useVercelIntegration } from './useVercelIntegration';

// type DisableProjectProps = {
//   isOpen: boolean;
//   onClose: () => void;
//   onConfirm: () => void;
// };

// const DisableVercel = ({ isOpen, onClose, onConfirm }: DisableProjectProps) => (
//   <AlertModal
//     className="w-1/3"
//     isOpen={isOpen}
//     onClose={onClose}
//     onSubmit={onConfirm}
//     title="Confirm disable"
//   >
//     <p className="px-6 pb-0 pt-4 dark:text-white">
//       Are you sure you want to disable Vercel Integration?
//     </p>
//   </AlertModal>
// );

export default function VercelProjects() {
  const { data } = useVercelIntegration();
  const router = useRouter();
  const { projects } = data;
  const [filter, setFilter] = useState('all');

  return (
    <div className="mt-8 flex flex-col">
      <div className="flex flex-row items-center justify-between">
        <div className="text-slate-500">
          Projects (<span className="mx-[2px]">{projects.length}</span>)
        </div>
        <div className="flex cursor-pointer flex-row items-center justify-between text-xs text-indigo-600">
          <NewButton
            onClick={() => router.refresh()}
            appearance="ghost"
            icon={<RiRefreshLine className="h-4 w-4" />}
            iconSide="left"
            label="Refresh list"
          />

          <Select
            value={{ id: 'all', name: 'All' }}
            onChange={(o) => setFilter(o.name)}
            label="Show"
            className="ml-4 h-6 rounded-sm bg-white text-xs leading-tight text-slate-500"
          >
            <Select.Button className="rounded-0 h-4">
              <span className="text-slate- pr-2 text-xs leading-tight text-slate-700 first-letter:capitalize">
                {filter}
              </span>
            </Select.Button>
            <Select.Options>
              {['all', 'disabled', 'enabled'].map((o, i) => {
                return (
                  <Select.Option key={`option-${i}`} option={{ id: o, name: o }}>
                    <span className="inline-flex w-full items-center justify-between gap-2">
                      <label className="text-sm lowercase first-letter:capitalize">{o}</label>
                    </span>
                  </Select.Option>
                );
              })}
            </Select.Options>
          </Select>
        </div>
      </div>
      {projects
        .filter((p) =>
          filter === 'all' ? true : filter === 'enabled' ? p.isEnabled : !p.isEnabled
        )
        .map((p, i) => (
          <Card
            key={`vercel-projects-${i}`}
            className="mt-4"
            accentPosition="left"
            accentColor="bg-indigo-400"
          >
            <Card.Content className="h-36 p-6">
              <div className="flex flex-row items-center justify-between">
                <div className="flex flex-col">
                  <div>
                    <Badge
                      kind="solid"
                      className={`h-6 ${
                        p.isEnabled ? 'bg-indigo-500 text-white' : 'bg-slate-200 text-slate-500'
                      }`}
                    >
                      {p.isEnabled ? 'enabled' : 'disabled'}
                    </Badge>
                  </div>
                  <div className="mt-4 flex flex-row items-center justify-start">
                    <div className="text-xl font-medium text-gray-900">{p.name}</div>
                    {p.ssoProtection?.deploymentType ===
                      VercelDeploymentProtection.ProdDeploymentURLsAndAllPreviews && (
                      <Tooltip>
                        <TooltipTrigger>
                          <RiInformationLine className="ml-2 h-4 w-4 cursor-pointer text-amber-500" />
                        </TooltipTrigger>
                        <TooltipContent className="rounded p-0">
                          <div className="border border-slate-200">
                            <div className="px-4 pt-2 text-sm font-medium text-slate-700">
                              Deployment protection is enabled
                            </div>
                            <div className="my-2 px-4 text-sm font-normal text-slate-500">
                              Inngest may not be able to communicate with your application by
                              default.
                            </div>
                            <div className="w-full bg-slate-200 px-4 py-2">
                              <Link href="https://www.inngest.com/docs/deploy/vercel#bypassing-deployment-protection">
                                Learn more
                              </Link>
                            </div>
                          </div>
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </div>
                  <div className="mt-2 text-base font-normal leading-snug text-slate-500">
                    {p.servePath}
                  </div>
                </div>
                <div>
                  <NewButton
                    appearance="outlined"
                    label="Configure"
                    href={`/settings/integrations/vercel/configure/${encodeURIComponent(p.id)}`}
                  />
                </div>
              </div>
            </Card.Content>
          </Card>
        ))}

      {/* <div className="mt-10 flex flex-col gap-4 border-t border-slate-200 py-7">
        <div className="text-lg font-medium text-gray-900">Disable Vercel integration</div>
        <div className="text-base font-normal leading-snug text-slate-600">
          This action disables API key and stops webhooks.
        </div>
        <div>
          {disable && (
            <DisableVercel
              isOpen={disable}
              onClose={() => setDisable(false)}
              onConfirm={async () => {
                //
                // TODO: implement this on the backend
              }}
            />
          )}
          <NewButton
            kind="danger"
            appearance="outlined"
            label="Disable Vercel"
            onClick={() => setDisable(true)}
          />
        </div>
      </div> */}
    </div>
  );
}
