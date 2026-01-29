import React from 'react';
import figma from '@figma/code-connect';

import { Input } from './Input';

figma.connect(
  Input,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=280%3A6471',
  {
    props: {
      disabled: figma.enum('state', {
        disabled: true,
      }),
      placeholder: figma.string('Text'),
    },
    example: (props) => <Input disabled={props.disabled} placeholder={props.placeholder} />,
  }
);
