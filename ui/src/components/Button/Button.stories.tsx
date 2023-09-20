import type { Meta, StoryObj } from '@storybook/react';

import { IconChevron } from '@/icons';
import Button from './Button';

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
      options: [null, <IconChevron />],
      control: { type: 'select' },
    }
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
    icon: <IconChevron />,
  },
};
export const PrimarySolidOnlyIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
    icon: <IconChevron />,
    label: null,
  },
};
export const PrimarySolidLoading: Story = {
  args: {
    kind: 'primary',
    appearance: 'solid',
    icon: <IconChevron />,
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
    icon: <IconChevron />,
  },
};
export const PrimaryOutlinedOnlyIcon: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
    icon: <IconChevron />,
    label: null,
  },
};
export const PrimaryOutlinedLoading: Story = {
  args: {
    kind: 'primary',
    appearance: 'outlined',
    icon: <IconChevron />,
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

export const PrimaryDisabled: Story = {
  args: {
    kind: 'primary',
    disabled: true,
  },
};

export const Default: Story = {
  args: {
    kind: 'default',
  },
};
export const DefaultSolid: Story = {
  args: {
    kind: 'default',
    appearance: 'solid',
  },
};
export const DefaultSolidWithIcon: Story = {
  args: {
    kind: 'default',
    appearance: 'solid',
    icon: <IconChevron />,
  },
};
export const DefaultSolidOnlyIcon: Story = {
  args: {
    kind: 'default',
    appearance: 'solid',
    icon: <IconChevron />,
    label: null,
  },
};
export const DefaultSolidLoading: Story = {
  args: {
    kind: 'default',
    appearance: 'solid',
    icon: <IconChevron />,
    loading: true,
    label: 'Loading...',
  },
};
export const DefaultSolidWithShortcut: Story = {
  args: {
    kind: 'default',
    appearance: 'solid',
    keys: ['A'],
  },
};
export const DefaultOutlined: Story = {
  args: {
    kind: 'default',
    appearance: 'outlined',
  },
};
export const DefaultOutlinedWithIcon: Story = {
  args: {
    kind: 'default',
    appearance: 'outlined',
    icon: <IconChevron />,
  },
};
export const DefaultOutlinedOnlyIcon: Story = {
  args: {
    kind: 'default',
    appearance: 'outlined',
    icon: <IconChevron />,
    label: null,
  },
};
export const DefaultOutlinedLoading: Story = {
  args: {
    kind: 'default',
    appearance: 'outlined',
    icon: <IconChevron />,
    loading: true,
    label: 'Loading...',
  },
};
export const DefaultOutlinedWithShortcut: Story = {
  args: {
    kind: 'default',
    appearance: 'outlined',
    keys: ['A'],
  },
};

export const DefaultDisabled: Story = {
  args: {
    kind: 'default',
    disabled: true,
  },
};

export const Success: Story = {
  args: {
    kind: 'success',
  },
};
export const SuccessSolid: Story = {
  args: {
    kind: 'success',
    appearance: 'solid',
  },
};
export const SuccessSolidWithIcon: Story = {
  args: {
    kind: 'success',
    appearance: 'solid',
    icon: <IconChevron />,
  },
};
export const SuccessSolidOnlyIcon: Story = {
  args: {
    kind: 'success',
    appearance: 'solid',
    icon: <IconChevron />,
    label: null,
  },
};
export const SuccessSolidLoading: Story = {
  args: {
    kind: 'success',
    appearance: 'solid',
    icon: <IconChevron />,
    loading: true,
    label: 'Loading...',
  },
};
export const SuccessSolidWithShortcut: Story = {
  args: {
    kind: 'success',
    appearance: 'solid',
    keys: ['A'],
  },
};
export const SuccessOutlined: Story = {
  args: {
    kind: 'success',
    appearance: 'outlined',
  },
};
export const SuccessOutlinedWithIcon: Story = {
  args: {
    kind: 'success',
    appearance: 'outlined',
    icon: <IconChevron />,
  },
};
export const SuccessOutlinedOnlyIcon: Story = {
  args: {
    kind: 'success',
    appearance: 'outlined',
    icon: <IconChevron />,
    label: null,
  },
};
export const SuccessOutlinedLoading: Story = {
  args: {
    kind: 'success',
    appearance: 'outlined',
    icon: <IconChevron />,
    loading: true,
    label: 'Loading...',
  },
};
export const SuccessOutlinedWithShortcut: Story = {
  args: {
    kind: 'success',
    appearance: 'outlined',
    keys: ['A'],
  },
};

export const SuccessDisabled: Story = {
  args: {
    kind: 'success',
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
    icon: <IconChevron />,
  },
};
export const DangerSolidOnlyIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
    icon: <IconChevron />,
    label: null,
  },
};
export const DangerSolidLoading: Story = {
  args: {
    kind: 'danger',
    appearance: 'solid',
    icon: <IconChevron />,
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
    icon: <IconChevron />,
  },
};
export const DangerOutlinedOnlyIcon: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
    icon: <IconChevron />,
    label: null,
  },
};
export const DangerOutlinedLoading: Story = {
  args: {
    kind: 'danger',
    appearance: 'outlined',
    icon: <IconChevron />,
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

export const DangerDisabled: Story = {
  args: {
    kind: 'danger',
    disabled: true,
  },
};

export const SmallSize: Story = {
  args: {
    kind: 'default',
    size: 'small',
  },
};

export const RegularSize: Story = {
  args: {
    kind: 'default',
    size: 'regular',
  },
};

export const LargeSize: Story = {
  args: {
    kind: 'default',
    size: 'large',
  },
};
