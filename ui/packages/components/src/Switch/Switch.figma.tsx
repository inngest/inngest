import React from 'react';
import figma from '@figma/code-connect';

import { Switch } from './Switch';

figma.connect(
  Switch,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=13%3A11359',
  {
    props: {
      disabled: figma.enum('State', {
        Disabled: true,
      }),
      checked: figma.enum('Status', {
        Checked: true,
      }),
    },
    example: (props) => <Switch disabled={props.disabled} checked={props.checked} />,
  }
);
