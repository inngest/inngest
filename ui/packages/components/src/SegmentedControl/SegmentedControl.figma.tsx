import React from 'react';
import figma from '@figma/code-connect';

import SegmentedControl from './SegmentedControl';

figma.connect(
  SegmentedControl,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=2565%3A1651',
  {
    props: {},
    example: () => (
      <SegmentedControl defaultValue="option1">
        <SegmentedControl.Button value="option1">Option 1</SegmentedControl.Button>
        <SegmentedControl.Button value="option2">Option 2</SegmentedControl.Button>
      </SegmentedControl>
    ),
  }
);
