import React from 'react';
import figma from '@figma/code-connect';

import { SlideOver } from './SlideOver';

figma.connect(
  SlideOver,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=2852%3A13144',
  {
    props: {},
    example: () => <SlideOver onClose={() => {}}>SlideOver content</SlideOver>,
  }
);
