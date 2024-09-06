'use client';

import Link from 'next/link';
import { Listbox } from '@headlessui/react';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import {
  RiArticleLine,
  RiBookReadLine,
  RiDiscordLine,
  RiExternalLinkLine,
  RiMailLine,
  RiQuestionLine,
  RiRoadMapLine,
} from '@remixicon/react';
import { useLocalStorage } from 'react-use';

import { useSystemStatus } from '@/app/(organization-active)/support/statusPage';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { pathCreator } from '@/utils/urls';
import SystemStatusIcon from './SystemStatusIcon';

export const Help = ({ collapsed }: { collapsed: boolean }) => {
  const { value: onBoardingFlow } = useBooleanFlag('onboarding-flow-cloud');
  const [_, setIsOnboardingWidgetOpen] = useLocalStorage('showOnboardingWidget');
  const status = useSystemStatus();

  return (
    <Listbox>
      <Listbox.Button className="w-full ring-0">
        <MenuItem
          collapsed={collapsed}
          text="Help and Feedback"
          icon={<RiQuestionLine className="h-[18px] w-[18px]" />}
        />
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase absolute -right-48 bottom-0 z-50 ml-8 w-[199px] gap-y-0.5 rounded border shadow ring-0 focus:outline-none">
          <Link href="https://www.inngest.com/docs?ref=support-center" target="_blank">
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="docs"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiExternalLinkLine className="text-subtle mr-2 h-4 w-4 " />
                <div>Inngest Documentation</div>
              </div>
            </Listbox.Option>
          </Link>
          <Link href="/support" target="_blank">
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="support"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiMailLine className="text-subtle mr-2 h-4 w-4" />
                <div>Support</div>
              </div>
            </Listbox.Option>
          </Link>
          <Link href="https://www.inngest.com/discord" target="_blank">
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle mx-2 my-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="discord"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiDiscordLine className="text-subtle mr-2 h-4 w-4" />
                <div>Join Discord</div>
              </div>
            </Listbox.Option>
          </Link>
          <hr />
          <Link href="https://roadmap.inngest.com/roadmap" target="_blank">
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="roadmap"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiRoadMapLine className="text-subtle mr-2 h-4 w-4" />
                <div>Inngest Roadmap</div>
              </div>
            </Listbox.Option>
          </Link>
          <Link href="/support" target="_blank">
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="status"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <SystemStatusIcon status={status} className="mx-0 mr-2 h-3.5 w-3.5" />
                <div>Status</div>
              </div>
            </Listbox.Option>
          </Link>
          <Link href="https://roadmap.inngest.com/changelog" target="_blank">
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle m-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="releaseNotes"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiArticleLine className="text-subtle mr-2 h-4 w-4" />
                <div>Release Notes</div>
              </div>
            </Listbox.Option>
          </Link>
          {onBoardingFlow && (
            <>
              <hr />
              <Link href={pathCreator.onboarding()} onClick={() => setIsOnboardingWidgetOpen(true)}>
                <Listbox.Option
                  className="text-subtle hover:bg-canvasSubtle m-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
                  value="onboardingGuide"
                >
                  <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                    <RiBookReadLine className="text-subtle mr-2 h-4 w-4" />
                    <div>Show onboarding guide</div>
                  </div>
                </Listbox.Option>
              </Link>
            </>
          )}
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
