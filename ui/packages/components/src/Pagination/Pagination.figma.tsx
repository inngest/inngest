import React from 'react';
import figma from '@figma/code-connect';

import { Pagination } from './Pagination';

figma.connect(
  Pagination,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=3597%3A546',
  {
    props: {},
    example: () => <Pagination currentPage={1} setCurrentPage={() => {}} totalPages={10} />,
  }
);
