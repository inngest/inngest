import { useState } from 'react';
import { RiArchive2Line, RiFirstAidKitLine, RiRefreshLine } from '@remixicon/react';
import type { Meta, StoryObj } from '@storybook/react';

import { Button } from '../Button';
import { Switch } from '../Switch';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from './DropdownMenu';
import { menuDangerItemClassName } from './menuStyles';

const meta = {
  title: 'Components/DropdownMenu',
  component: DropdownMenu,
  parameters: { layout: 'centered' },
} satisfies Meta<typeof DropdownMenu>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Actions: Story = {
  render: () => (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button label="App actions" kind="secondary" />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-60">
        <DropdownMenuItem>
          <RiRefreshLine className="text-muted h-4 w-4" />
          Resync app
        </DropdownMenuItem>
        <DropdownMenuItem>
          <RiFirstAidKitLine className="text-muted h-4 w-4" />
          Check app health
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem className={menuDangerItemClassName}>
          <RiArchive2Line className="h-4 w-4" />
          Archive app
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  ),
};

export const DisabledAndShortcuts: Story = {
  render: () => (
    <DropdownMenu>
      <DropdownMenuTrigger>Query actions</DropdownMenuTrigger>
      <DropdownMenuContent className="w-72">
        <DropdownMenuItem>
          Format SQL<span className="text-muted ml-auto">⇧⌥F</span>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem disabled>
          Copy<span className="text-muted ml-auto">⌘C</span>
        </DropdownMenuItem>
        <DropdownMenuItem>
          Paste<span className="text-muted ml-auto">⌘V</span>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem>
          Run query<span className="text-muted ml-auto">⌘↵</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  ),
};

function ToggleMenu() {
  const [enabled, setEnabled] = useState(true);
  return (
    <DropdownMenu>
      <DropdownMenuTrigger>Display settings</DropdownMenuTrigger>
      <DropdownMenuContent className="w-72">
        <DropdownMenuItem className="justify-between">
          <div>
            <div>Auto Refresh</div>
            <div className="text-muted text-xs">Refreshes data every 5 seconds</div>
          </div>
          <Switch
            aria-label="Auto Refresh"
            checked={enabled}
            onClick={(event) => {
              event.stopPropagation();
              setEnabled(!enabled);
            }}
          />
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export const ToggleWithDescription: Story = { render: () => <ToggleMenu /> };

export const LongLabelsAndScrolling: Story = {
  render: () => (
    <DropdownMenu>
      <DropdownMenuTrigger>More actions</DropdownMenuTrigger>
      <DropdownMenuContent className="max-h-64 w-60 overflow-y-auto">
        <DropdownMenuItem>
          A longer action label that needs to wrap without clipping
        </DropdownMenuItem>
        {Array.from({ length: 20 }, (_, index) => (
          <DropdownMenuItem key={index}>Action {index + 1}</DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  ),
};
