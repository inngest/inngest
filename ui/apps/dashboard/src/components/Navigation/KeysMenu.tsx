import {
  headlessMenuItemClassName,
  menuPanelClassName,
} from '@inngest/components/DropdownMenu/menuStyles';
import { cn } from '@inngest/components/utils/classNames';
import { Listbox } from '@headlessui/react';
import { Button } from '@inngest/components/Button';
import { RiKey2Line } from '@remixicon/react';

import type { Environment as EnvType } from '@/utils/environments';
import { Link } from '@tanstack/react-router';
import { pathCreator } from '@/utils/urls';

export default function KeysMenu({
  activeEnv,
  collapsed,
}: {
  activeEnv: EnvType;
  collapsed: boolean;
}) {
  return (
    <Listbox>
      <Listbox.Button as="div" className="group">
        <Button
          kind="secondary"
          appearance={collapsed ? 'ghost' : 'outlined'}
          size="medium"
          icon={<RiKey2Line className="fill-subtle w-[18px]" />}
          className="group-data-[headlessui-state=open]:border-primary-moderate"
        />
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options
          className={cn(
            menuPanelClassName,
            'dropdown-focus absolute left-0 z-50 ml-1 w-[137px]',
          )}
        >
          <Link to={pathCreator.keys({ envSlug: activeEnv.slug })}>
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="eventKeys"
            >
              Event keys
            </Listbox.Option>
          </Link>

          <Link to={pathCreator.signingKeys({ envSlug: activeEnv.slug })}>
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="signingKeys"
            >
              Signing keys
            </Listbox.Option>
          </Link>
        </Listbox.Options>
      </div>
    </Listbox>
  );
}
