import React from 'react';
import figma from '@figma/code-connect';

import { Popover } from './Popover';

figma.connect(
  Popover,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=1176%3A4755',
  {
    props: {},
    example: () => <Popover>Popover content</Popover>,
  }
);
