'use client';

import dynamic from 'next/dynamic';
import { Listbox } from '@headlessui/react';
import { Pill } from '@inngest/components/Pill/Pill';

import { useFeatureFlags } from '@/hooks/useFeatureFlags';

const ModeSwitch = dynamic(() => import('@inngest/components/ThemeMode/ModeSwitch'), {
  ssr: false,
});

export const ProfileMenu = ({ children }: React.PropsWithChildren) => {
  const { featureFlags } = useFeatureFlags();
  const isThemeModeSwitchEnabled = featureFlags.FEATURE_THEME_MODE;

  return (
    <Listbox>
      <Listbox.Button className="w-full cursor-pointer ring-0">{children}</Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase border-muted shadow-primary absolute -right-48 bottom-4 z-50 ml-8 w-[199px] rounded border ring-0 focus:outline-none">
          <>
            <Listbox.Option
              disabled
              value="themeMode"
              className="text-muted mx-2 my-2 flex h-8 items-center justify-between px-2 text-[13px]"
            >
              <div>Theme</div>
              {isThemeModeSwitchEnabled ? <ModeSwitch /> : <Pill>Coming soon</Pill>}
            </Listbox.Option>

            <hr className="border-subtle" />
          </>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
