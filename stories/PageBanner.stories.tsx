import React from "react";
import { ComponentStory, ComponentMeta } from "@storybook/react";

import PageBanner from "../shared/PageBanner";

export default {
  title: "PageBanner",
  component: PageBanner,
} as ComponentMeta<typeof PageBanner>;

const Template: ComponentStory<typeof PageBanner> = (args) => (
  <PageBanner {...args} />
);

// More on args: https://storybook.js.org/docs/react/writing-stories/args
export const Default = Template.bind({});
Default.args = {
  href: "#goto-some-page",
  children:
    "Introducing the Inngest CLI: build, test, and ship serverless functions locally ›",
};
