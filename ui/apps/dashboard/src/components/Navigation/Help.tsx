import {
  headlessMenuItemClassName,
  menuPanelClassName,
  menuSeparatorClassName,
} from '@inngest/components/DropdownMenu/menuStyles';
import { cn } from '@inngest/components/utils/classNames';
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

import { pathCreator } from '@/utils/urls';
import useOnboardingStep from '../Onboarding/useOnboardingStep';
import SystemStatusIcon from './SystemStatusIcon';
import { Link } from '@tanstack/react-router';
import { useSystemStatus } from '../Support/SystemStatus';

export const Help = ({
  collapsed,
  showWidget,
}: {
  collapsed: boolean;
  showWidget: () => void;
}) => {
  const { nextStep, lastCompletedStep } = useOnboardingStep();
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
        <Listbox.Options
          className={cn(
            menuPanelClassName,
            'dropdown-focus absolute -right-48 bottom-0 z-50 ml-8 w-[199px]',
          )}
        >
          <a
            href="https://www.inngest.com/docs?ref=support-center"
            target="_blank"
          >
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="docs"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiExternalLinkLine className="text-muted mr-2 h-4 w-4 " />
                <div>Inngest Documentation</div>
              </div>
            </Listbox.Option>
          </a>
          <a href="https://support.inngest.com" target="_blank">
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="support"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiMailLine className="text-muted mr-2 h-4 w-4" />
                <div>Support</div>
              </div>
            </Listbox.Option>
          </a>
          <a href="https://www.inngest.com/discord" target="_blank">
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="discord"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiDiscordLine className="text-muted mr-2 h-4 w-4" />
                <div>Join Discord</div>
              </div>
            </Listbox.Option>
          </a>
          <hr className={menuSeparatorClassName} />
          <a href="https://roadmap.inngest.com/roadmap" target="_blank">
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="roadmap"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiRoadMapLine className="text-muted mr-2 h-4 w-4" />
                <div>Inngest Roadmap</div>
              </div>
            </Listbox.Option>
          </a>
          <a href="https://status.inngest.com" target="_blank">
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="status"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <SystemStatusIcon
                  status={status}
                  className="mx-0 mr-2 h-3.5 w-3.5"
                />
                <div>Status</div>
              </div>
            </Listbox.Option>
          </a>
          <a href="https://www.inngest.com/changelog" target="_blank">
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="releaseNotes"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiArticleLine className="text-muted mr-2 h-4 w-4" />
                <div>Release Notes</div>
              </div>
            </Listbox.Option>
          </a>
          <hr className={menuSeparatorClassName} />
          <Link
            to={pathCreator.onboardingSteps({
              step: nextStep ? nextStep.name : lastCompletedStep?.name,
              ref: 'app-navbar-help',
            })}
            onClick={() => showWidget()}
          >
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="onboardingGuide"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiBookReadLine className="text-muted mr-2 h-4 w-4" />
                <div>Show onboarding guide</div>
              </div>
            </Listbox.Option>
          </Link>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
