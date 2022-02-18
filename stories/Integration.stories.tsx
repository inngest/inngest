import React from 'react';
import { ComponentStory, ComponentMeta } from '@storybook/react';

import Integration, { IntegrationType } from '../shared/Integration';

export default {
  title: 'Integration',
  component: Integration,
} as ComponentMeta<typeof Integration>;

// More on component templates: https://storybook.js.org/docs/react/writing-stories/introduction#using-args
const Template: ComponentStory<typeof Integration> = (args) => <Integration {...args} />;

// More on args: https://storybook.js.org/docs/react/writing-stories/args
export const Stripe = Template.bind({});
Stripe.args = {
  name: "Stripe",
  logo: "/integrations/stripe.svg",
  category: "Payments & Billing",
  type: [IntegrationType.EVENTS],
};
