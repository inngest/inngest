import React from 'react';
import figma from '@figma/code-connect';

import { MenuItem } from './MenuItem';

figma.connect(
  MenuItem,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=1477%3A15367',
  {
    props: {
      icon: figma.instance('Menu-icon'),
      collapsed: figma.boolean('Collapsed?'),
    },
    example: (props) => <MenuItem text="Menu Item" icon={props.icon} collapsed={props.collapsed} />,
  }
);
