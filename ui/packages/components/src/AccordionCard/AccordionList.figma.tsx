import React from 'react';
import figma from '@figma/code-connect';

import { AccordionList } from './AccordionList';

figma.connect(
  AccordionList,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=2943%3A9961',
  {
    props: {},
    example: () => <AccordionList type="trigger" />,
  }
);
