import React from 'react';
import figma from '@figma/code-connect';

import { Tooltip, TooltipContent, TooltipTrigger } from './Tooltip';

figma.connect(
  Tooltip,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=1156%3A3800',
  {
    props: {},
    example: () => (
      <Tooltip>
        <TooltipTrigger>Hover me</TooltipTrigger>
        <TooltipContent>Tooltip content</TooltipContent>
      </Tooltip>
    ),
  }
);
