import React from 'react';
import { ComponentStory, ComponentMeta } from '@storybook/react';

import Button from '../shared/Button';

export default {
  title: 'Button',
  component: Button,
} as ComponentMeta<typeof Button>;

// More on component templates: https://storybook.js.org/docs/react/writing-stories/introduction#using-args
const Template: ComponentStory<typeof Button> = (args) => <Button {...args} />;

// More on args: https://storybook.js.org/docs/react/writing-stories/args
export const Primary = Template.bind({});
Primary.args = {
  kind: "primary",
  children: 'Start building',
};

export const Outline = Template.bind({});
Outline.args = {
  children: 'Explore docs →',
  kind: "outline"
};

export const OutlineLink = Template.bind({});
OutlineLink.args = {
  children: 'Explore docs →',
  href: "https://www.example.com",
  kind: "outline"
};
