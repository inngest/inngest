import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import type { Meta, StoryObj } from '@storybook/react';

import { Button } from './index';

const meta = {
  title: 'Components/Button',
  component: Button,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    label: 'Click me',
  },
  argTypes: {
    keys: {
      options: [[], ['↵'], ['⌘', 'A']],
      control: { type: 'select' },
    },
    icon: {
      options: [null, <FunctionsIcon />],
      control: { type: 'select' },
    },
  },
} satisfies Meta<typeof Button>;

export default meta;

type Story = StoryObj<typeof Button>;

export const PrimarySolid: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
  },
};
export const PrimarySolidWithIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
    icon: <FunctionsIcon />,
    iconSide: 'left',
  },
};
export const PrimarySolidOnlyIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
    icon: <FunctionsIcon />,
    label: null,
  },
};
export const PrimarySolidLoading: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
    icon: <FunctionsIcon />,
    loading: true,
    label: 'Loading...',
  },
};
export const PrimarySolidWithShortcut: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
    keys: ['A'],
  },
};
export const PrimarySolidDisabled: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
    disabled: true,
  },
};
export const PrimaryOutlined: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
  },
};
export const PrimaryOutlinedWithIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
    icon: <FunctionsIcon />,
    iconSide: 'left',
  },
};
export const PrimaryOutlinedOnlyIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
    icon: <FunctionsIcon />,
    label: null,
  },
};
export const PrimaryOutlinedLoading: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
    icon: <FunctionsIcon />,
    loading: true,
    label: 'Loading...',
  },
};
export const PrimaryOutlinedWithShortcut: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
    keys: ['A'],
  },
};

export const PrimaryOutlinedDisabled: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
    disabled: true,
  },
};

export const PrimaryGhost: Story = {
  args: {
    kind: 'primary',
    appearance: 'ghost',
  },
};
export const PrimaryGhostWithIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'ghost',
    icon: <FunctionsIcon />,
    iconSide: 'left',
  },
};
export const PrimaryGhostOnlyIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'ghost',
    icon: <FunctionsIcon />,
    label: null,
  },
};
export const PrimaryGhostLoading: Story = {
  args: {
    kind: 'primary',
    appearance: 'ghost',
    icon: <FunctionsIcon />,
    loading: true,
    label: 'Loading...',
  },
};
export const PrimaryGhostWithShortcut: Story = {
  args: {
    kind: 'primary',
    appearance: 'ghost',
    keys: ['A'],
  },
};

export const PrimaryGhostDisabled: Story = {
  args: {
    kind: 'primary',
    appearance: 'ghost',
    disabled: true,
  },
};

export const SecondaryOutlined: Story = {
  args: {
    kind: 'secondary',
    appearance: 'outlined',
  },
};
export const SecondaryOutlinedWithIcon: Story = {
  args: {
    kind: 'secondary',
    appearance: 'outlined',
    icon: <FunctionsIcon />,
    iconSide: 'left',
  },
};
export const SecondaryOutlinedOnlyIcon: Story = {
  args: {
    kind: 'secondary',
    appearance: 'outlined',
    icon: <FunctionsIcon />,
    label: null,
  },
};
export const SecondaryOutlinedLoading: Story = {
  args: {
    kind: 'secondary',
    appearance: 'outlined',
    icon: <FunctionsIcon />,
    loading: true,
    label: 'Loading...',
  },
};
export const SecondaryOutlinedWithShortcut: Story = {
  args: {
    kind: 'secondary',
    appearance: 'outlined',
    keys: ['A'],
  },
};
export const SecondaryOutlinedDisabled: Story = {
  args: {
    kind: 'secondary',
    appearance: 'outlined',
    disabled: true,
  },
};
export const SecondaryGhost: Story = {
  args: {
    kind: 'secondary',
    appearance: 'ghost',
  },
};
export const SecondaryGhostWithIcon: Story = {
  args: {
    kind: 'secondary',
    appearance: 'ghost',
    icon: <FunctionsIcon />,
    iconSide: 'left',
  },
};
export const SecondaryGhostOnlyIcon: Story = {
  args: {
    kind: 'secondary',
    appearance: 'ghost',
    icon: <FunctionsIcon />,
    label: null,
  },
};
export const SecondaryGhostLoading: Story = {
  args: {
    kind: 'secondary',
    appearance: 'ghost',
    icon: <FunctionsIcon />,
    loading: true,
    label: 'Loading...',
  },
};
export const SecondaryGhostWithShortcut: Story = {
  args: {
    kind: 'secondary',
    appearance: 'ghost',
    keys: ['A'],
  },
};

export const SecondaryGhostDisabled: Story = {
  args: {
    kind: 'secondary',
    appearance: 'ghost',
    disabled: true,
  },
};

export const DangerSolid: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
  },
};
export const DangerSolidWithIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
    icon: <FunctionsIcon />,
    iconSide: 'left',
  },
};
export const DangerSolidOnlyIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
    icon: <FunctionsIcon />,
    label: null,
  },
};
export const DangerSolidLoading: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
    icon: <FunctionsIcon />,
    loading: true,
    label: 'Loading...',
  },
};
export const DangerSolidWithShortcut: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
    keys: ['A'],
  },
};
export const DangerSolidDisabled: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
    disabled: true,
  },
};
export const DangerOutlined: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
  },
};
export const DangerOutlinedWithIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
    icon: <FunctionsIcon />,
    iconSide: 'left',
  },
};
export const DangerOutlinedOnlyIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
    icon: <FunctionsIcon />,
    label: null,
  },
};
export const DangerOutlinedLoading: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
    icon: <FunctionsIcon />,
    loading: true,
    label: 'Loading...',
  },
};
export const DangerOutlinedWithShortcut: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
    keys: ['A'],
  },
};
export const DangerOutlinedDisabled: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
    disabled: true,
  },
};
export const DangerGhost: Story = {
  args: {
    kind: 'danger',
    appearance: 'ghost',
  },
};
export const DangerGhostWithIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'ghost',
    icon: <FunctionsIcon />,
    iconSide: 'left',
  },
};
export const DangerGhostOnlyIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'ghost',
    icon: <FunctionsIcon />,
    label: null,
  },
};
export const DangerGhostLoading: Story = {
  args: {
    kind: 'danger',
    appearance: 'ghost',
    icon: <FunctionsIcon />,
    loading: true,
    label: 'Loading...',
  },
};
export const DangerGhostWithShortcut: Story = {
  args: {
    kind: 'danger',
    appearance: 'ghost',
    keys: ['A'],
  },
};

export const DangerGhostDisabled: Story = {
  args: {
    kind: 'danger',
    appearance: 'ghost',
    disabled: true,
  },
};

export const SmallSize: Story = {
  args: {
    kind: 'primary',
    size: 'small',
  },
};

export const RegularSize: Story = {
  args: {
    kind: 'primary',
    size: 'medium',
  },
};

export const LargeSize: Story = {
  args: {
    kind: 'primary',
    size: 'large',
  },
};
