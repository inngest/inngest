import React from 'react';
import figma from '@figma/code-connect';

import { Modal } from './Modal';

figma.connect(
  Modal,
  'https://www.figma.com/design/3qz5WWvj0MCnivg7fcaizC/Earl-v1?node-id=1568%3A12092',
  {
    props: {},
    example: () => (
      <Modal isOpen={true} onClose={() => {}}>
        Modal content
      </Modal>
    ),
  }
);
