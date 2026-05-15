import React from 'react';
import figma from '@figma/code-connect';

import { Alert } from './Alert';

figma.connect(
  Alert,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=599%3A4469',
  {
    props: {
      severity: figma.enum('type?', {
        info: 'info',
        warning: 'warning',
        error: 'error',
        success: 'success',
      }),
    },
    example: (props) => <Alert severity={props.severity}>Alert message content</Alert>,
  }
);
