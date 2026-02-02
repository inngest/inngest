import React from 'react';
import figma from '@figma/code-connect';

import { InngestLogo } from './InngestLogo';

figma.connect(
  InngestLogo,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=1477%3A15352',
  {
    props: {},
    example: () => <InngestLogo />,
  }
);
