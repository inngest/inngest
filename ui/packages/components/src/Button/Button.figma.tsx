import React from 'react';
import figma from '@figma/code-connect';

import { Button } from './Button';

figma.connect(
  Button,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=312%3A1107',
  {
    props: {
      size: figma.enum('size', {
        Small: 'small',
        Medium: 'medium',
        Large: 'large',
      }),
      kind: figma.enum('variant', {
        Primary: 'primary',
        Secondary: 'secondary',
        Danger: 'danger',
      }),
      appearance: figma.enum('type', {
        Solid: 'solid',
        Outline: 'outlined',
        Ghost: 'ghost',
      }),
      label: figma.string('Text'),
      icon: figma.instance('iconOnly'),
      loading: figma.enum('state', {
        Loading: true,
      }),
      disabled: figma.enum('state', {
        Disabled: true,
      }),
    },
    example: (props) => (
      <Button
        kind={props.kind}
        appearance={props.appearance}
        size={props.size}
        label={props.label}
        icon={props.icon}
        loading={props.loading}
        disabled={props.disabled}
      />
    ),
  }
);
