import React from "react"
import { ComponentStory, ComponentMeta } from "@storybook/react"

import Banner, { CheckBanner } from "../shared/Banner"

export default {
  title: "Banner",
  component: Banner,
} as ComponentMeta<typeof Banner>

const Template: ComponentStory<typeof Banner> = (args) => <Banner {...args} />
const CheckTemplate: ComponentStory<typeof CheckBanner> = (args) => (
  <CheckBanner {...args} />
)

// More on args: https://storybook.js.org/docs/react/writing-stories/args
export const Default = Template.bind({})
Default.args = {
  children: <p>Hi</p>,
}

export const Check = CheckTemplate.bind({})
Check.args = {
  list: ["Developer CLI", "Auto-gen'd types", "TypeScript"],
}
