import React from 'react';
import figma from '@figma/code-connect';

import { Checkbox } from './Checkbox';

figma.connect(
  Checkbox,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=13%3A12718',
  {
    props: {
      disabled: figma.enum('State', {
        Disabled: true,
      }),
      checked: figma.enum('Status', {
        Checked: true,
      }),
    },
    example: (props) => <Checkbox disabled={props.disabled} checked={props.checked} />,
  }
);
