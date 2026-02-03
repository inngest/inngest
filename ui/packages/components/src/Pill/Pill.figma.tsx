import React from 'react';
import figma from '@figma/code-connect';

import { Pill } from './Pill';

figma.connect(
  Pill,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=1397%3A6085',
  {
    props: {
      appearance: figma.enum('Type', {
        Solid: 'solid',
        Outline: 'outlined',
        'Solid-bright': 'solidBright',
      }),
      kind: figma.enum('Variant', {
        Default: 'default',
        Primary: 'primary',
        Information: 'info',
        Warning: 'warning',
        Error: 'error',
      }),
    },
    example: (props) => (
      <Pill kind={props.kind} appearance={props.appearance}>
        Label
      </Pill>
    ),
  }
);
