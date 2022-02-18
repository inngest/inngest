import React from "react"
import { ComponentStory, ComponentMeta } from "@storybook/react"

import IconList from "../shared/IconList"

import CheckIcon from "../shared/Icons/Check"
import WorkflowIcon from "../shared/Icons/Workflow"
import LanguageIcon from "../shared/Icons/Language"
import LightningIcon from "../shared/Icons/Lightning"

export default {
  title: "IconList",
  component: IconList,
} as ComponentMeta<typeof IconList>

const Template: ComponentStory<typeof IconList> = (args) => (
  <IconList {...args} />
)

const checklistItems = [
  {
    icon: CheckIcon,
    text: "Developer CLI",
  },
  {
    icon: CheckIcon,
    text: "Auto-gen'd types & schemas",
  },
  {
    icon: CheckIcon,
    text: "Retries & replays built in",
  },
]

export const Default = Template.bind({})
Default.args = {
  items: checklistItems,
}

export const Vertical = Template.bind({})
Vertical.args = {
  direction: "vertical",
  items: checklistItems,
}

export const Small = Template.bind({})
Small.args = {
  direction: "vertical",
  size: "small",
  items: checklistItems,
}

const mixedIconItems = [
  {
    icon: WorkflowIcon,
    quantity: "5",
    text: "Workflows",
  },
  {
    icon: LanguageIcon,
    text: (
      <>
        <strong>25</strong> Functions
      </>
    ),
  },
  {
    icon: LightningIcon,
    text: "Resources",
  },
  {
    icon: CheckIcon,
    text: "Retries & replays built in",
  },
]

export const MixedIcons = Template.bind({})
MixedIcons.args = {
  direction: "vertical",
  items: mixedIconItems,
}

export const WithoutCircles = Template.bind({})
WithoutCircles.args = {
  direction: "vertical",
  circles: false,
  items: mixedIconItems,
}
