import React from 'react';
import figma from '@figma/code-connect';

import { CodeBlock } from './CodeBlock';

figma.connect(
  CodeBlock,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=1469%3A1942',
  {
    props: {},
    example: () => (
      <CodeBlock
        tab={{
          label: 'Code',
          content: "const example = 'code';",
        }}
      />
    ),
  }
);
