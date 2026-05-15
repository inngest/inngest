import React from 'react';
import figma from '@figma/code-connect';

import { Select } from './Select';

figma.connect(
  Select,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=2905%3A5510',
  {
    props: {
      size: figma.enum('size', {
        small: 'small',
      }),
      disabled: figma.enum('state', {
        disabled: true,
      }),
    },
    example: (props) => (
      <Select size={props.size} disabled={props.disabled} onChange={() => {}} multiple={false}>
        <option value="1">Option 1</option>
        <option value="2">Option 2</option>
      </Select>
    ),
  }
);
