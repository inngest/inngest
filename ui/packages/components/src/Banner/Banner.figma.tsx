import React from 'react';
import figma from '@figma/code-connect';

import { Banner } from './Banner';

figma.connect(
  Banner,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=1264%3A8277',
  {
    props: {
      severity: figma.enum('Type', {
        Info: 'info',
        Success: 'success',
        Warning: 'warning',
        Error: 'error',
      }),
    },
    example: (props) => <Banner severity={props.severity}>Banner message content</Banner>,
  }
);
