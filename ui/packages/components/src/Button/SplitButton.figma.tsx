import React from 'react';
import figma from '@figma/code-connect';

import { SplitButton } from './SplitButton';

figma.connect(
  SplitButton,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=2789%3A8308',
  {
    props: {},
    example: () => <SplitButton left={<button>Action</button>} right={<button>More</button>} />,
  }
);
