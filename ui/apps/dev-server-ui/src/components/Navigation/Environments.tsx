'use client';

import { Button } from '@inngest/components/Button';
import { RiExpandUpDownLine } from '@remixicon/react';

export const EnvironmentMenu = ({ collapsed }: { collapsed: boolean }) => {
  return (
    <Button
      disabled={true}
      kind="secondary"
      appearance="outlined"
      {...(collapsed ? {} : { icon: <RiExpandUpDownLine className="text-disabled h-4 w-4" /> })}
      {...(collapsed ? {} : { iconSide: 'right' })}
      label={collapsed ? 'DV' : 'Development'}
      className={`text-disabled text-sm ${
        collapsed ? 'w-8 justify-center' : 'w-full justify-between'
      }`}
    />
  );
};
