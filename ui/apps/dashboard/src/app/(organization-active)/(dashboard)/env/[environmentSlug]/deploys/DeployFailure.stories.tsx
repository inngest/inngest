import type { Meta, StoryObj } from '@storybook/react';

import { BaseWrapper } from '@/app/baseWrapper';
import { DeployFailure } from './DeployFailure';
import { registrationErrorCodes } from './utils';

const meta: Meta<typeof DeployFailure> = {
  argTypes: {
    errorCode: {
      // Needed because Storybook's automatically generated options includes the
      // string "undefined" instead of the undefined literal.
      options: [...registrationErrorCodes, undefined],
    },
  },
  args: {
    errorCode: 'unauthorized',
    headers: {
      Connection: ['keep-alive'],
      'Content-Length': ['151'],
      'Content-Security-Policy': ["default-src 'none'"],
      'Content-Type': ['text/html; charset=utf-8'],
      Date: ['Tue, 04 Jul 2023 02:10:51 GMT'],
      'Keep-Alive': ['timeout=5'],
      'X-Content-Type-Options': ['nosniff'],
      'X-Powered-By': ['Express'],
    },
    statusCode: 420,
  },
  decorators: [
    (Story) => {
      return (
        <BaseWrapper>
          <Story />
        </BaseWrapper>
      );
    },
  ],
  component: DeployFailure,
  tags: ['autodocs'],
  title: 'DeployFailure',
};

export default meta;
type Story = StoryObj<typeof DeployFailure>;

export const Primary: Story = {};
