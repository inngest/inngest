import React from 'react';
import figma from '@figma/code-connect';

import { TableFilter } from './Filter';

figma.connect(
  TableFilter,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=3809%3A3147',
  {
    props: {},
    example: () => <TableFilter options={[]} setColumnVisibility={() => {}} />,
  }
);
