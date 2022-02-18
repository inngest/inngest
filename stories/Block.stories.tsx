import React from "react"
import { ComponentStory, ComponentMeta } from "@storybook/react"

import Block from "../shared/Block"

export default {
  title: "Block",
  component: Block,
} as ComponentMeta<typeof Block>

const Template: ComponentStory<typeof Block> = (args) => <Block {...args} />

export const Default = Template.bind({})
Default.args = {
  children: [<h3>Title</h3>, <p>Some description copy</p>],
}

export const Primary = Template.bind({})
Primary.args = {
  color: "primary",
  children: [<h3>Title</h3>, <p>Some description copy</p>],
}
