import React from 'react';
import { ComponentStory, ComponentMeta } from '@storybook/react';

import Code from '../shared/Code';

export default {
  title: 'Code',
  component: Code,
} as ComponentMeta<typeof Code>;

// More on component templates: https://storybook.js.org/docs/react/writing-stories/introduction#using-args
const Template: ComponentStory<typeof Code> = (args) => <Code {...args} />;

// More on args: https://storybook.js.org/docs/react/writing-stories/args
export const SendEvents = Template.bind({});
SendEvents.args = {
  code: {
    cURL: `curl -X POST ”https://inn.gs/e/test-key-goes-here-bjm8xj6nji0vzzu0l1k” \\
  -d '{"name": "test.event", "data": { "email": “gob@bluth-dev.com” } }'}`,
    Go: `// TODO`,
  },
};
