import React from 'react';
import { ComponentMeta, ComponentStory } from '@storybook/react';

import VideoPlayer from '../shared/legacy/VideoPlayer';

export default {
  title: 'VideoPlayer',
  component: VideoPlayer,
} as ComponentMeta<typeof VideoPlayer>;

const Template: ComponentStory<typeof VideoPlayer> = (args) => (
  <div style={{ maxWidth: '800px' }}>
    <VideoPlayer {...args} />
  </div>
);

export const Default = Template.bind({});
Default.args = {
  src: '/assets/homepage/init-run-deploy-2022-04-20.mp4',
  duration: 53,
  chapters: [
    {
      name: 'Build',
      start: 0,
    },
    {
      name: 'Test',
      start: 20,
    },
    {
      name: 'Deploy',
      start: 29.1,
    },
  ],
};

export const NoChapters = Template.bind({});
NoChapters.args = {
  src: '/assets/homepage/init-run-deploy-2022-04-20.mp4',
  duration: 53,
};
