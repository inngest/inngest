import React from 'react';
import figma from '@figma/code-connect';

import { BreadCrumb } from './BreadCrumb';

figma.connect(
  BreadCrumb,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=319%3A10367',
  {
    props: {},
    example: () => <BreadCrumb path={[{ text: 'Home' }, { text: 'Page' }]} />,
  }
);
