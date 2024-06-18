import { IconFunction } from '@inngest/components/icons/Function';
import type { Meta, StoryObj } from '@storybook/react';

import { NewButton } from './index';

const meta = {
  title: 'Components/Button',
  component: NewButton,
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
      options: [null, <IconFunction />],
      control: { type: 'select' },
    },
  },
} satisfies Meta<typeof NewButton>;

export default meta;

type Story = StoryObj<typeof NewButton>;

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
    icon: <IconFunction />,
  },
};
export const PrimarySolidOnlyIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
    icon: <IconFunction />,
    label: null,
  },
};
export const PrimarySolidLoading: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
    icon: <IconFunction />,
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
    icon: <IconFunction />,
  },
};
export const PrimaryOutlinedOnlyIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
    icon: <IconFunction />,
    label: null,
  },
};
export const PrimaryOutlinedLoading: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
    icon: <IconFunction />,
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
    icon: <IconFunction />,
  },
};
export const SecondaryOutlinedOnlyIcon: Story = {
  args: {
    kind: 'secondary',
    appearance: 'outlined',
    icon: <IconFunction />,
    label: null,
  },
};
export const SecondaryOutlinedLoading: Story = {
  args: {
    kind: 'secondary',
    appearance: 'outlined',
    icon: <IconFunction />,
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

export const Danger: Story = {
  args: {
    kind: 'danger',
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
    icon: <IconFunction />,
  },
};
export const DangerSolidOnlyIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
    icon: <IconFunction />,
    label: null,
  },
};
export const DangerSolidLoading: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
    icon: <IconFunction />,
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
    icon: <IconFunction />,
  },
};
export const DangerOutlinedOnlyIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
    icon: <IconFunction />,
    label: null,
  },
};
export const DangerOutlinedLoading: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
    icon: <IconFunction />,
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
