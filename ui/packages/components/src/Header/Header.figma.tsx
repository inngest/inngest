import React from 'react';
import figma from '@figma/code-connect';

import { Header } from './Header';

figma.connect(
  Header,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=1477%3A15646',
  {
    props: {},
    example: () => <Header breadcrumb={<span>Breadcrumb</span>} />,
  }
);
