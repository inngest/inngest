import React from 'react';
import figma from '@figma/code-connect';

import { Tabs } from './Tabs';

figma.connect(
  Tabs,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=3913%3A8678',
  {
    props: {},
    example: () => (
      <Tabs
        tabs={[
          { id: 'tab1', label: 'Tab 1', content: <div>Content 1</div> },
          { id: 'tab2', label: 'Tab 2', content: <div>Content 2</div> },
        ]}
      />
    ),
  }
);
