import React from 'react';
import figma from '@figma/code-connect';

import ProgressBar from './ProgressBar';

figma.connect(
  ProgressBar,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=2840%3A12014',
  {
    props: {},
    example: () => <ProgressBar limit={100} value={50} />,
  }
);
