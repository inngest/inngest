import React from 'react';
import figma from '@figma/code-connect';

import { RangePicker } from './RangePicker';

figma.connect(
  RangePicker,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=609%3A3984',
  {
    props: {
      type: figma.enum('Type', {
        Relative: 'reset',
      }),
    },
    example: (props) => <RangePicker type={props.type} onChange={() => {}} />,
  }
);
