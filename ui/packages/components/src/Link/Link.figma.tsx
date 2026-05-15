import React from 'react';
import figma from '@figma/code-connect';

import { Link } from './Link';

figma.connect(
  Link,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=13%3A10701',
  {
    props: {
      size: figma.enum('Size', {
        Small: 'small',
        Medium: 'medium',
      }),
      label: figma.string('Text'),
      iconBefore: figma.instance('iconBefore'),
      iconAfter: figma.instance('iconAfter'),
    },
    example: (props) => (
      <Link href="#" size={props.size} iconBefore={props.iconBefore} iconAfter={props.iconAfter}>
        {props.label}
      </Link>
    ),
  }
);
