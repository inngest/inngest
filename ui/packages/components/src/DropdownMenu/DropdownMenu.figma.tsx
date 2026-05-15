import React from 'react';
import figma from '@figma/code-connect';

import { DropdownMenu } from './DropdownMenu';

figma.connect(
  DropdownMenu,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=596%3A4055',
  {
    props: {},
    example: () => <DropdownMenu />,
  }
);
