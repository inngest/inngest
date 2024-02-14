import React, { type PropsWithChildren } from 'react';
import styled from '@emotion/styled';

import { Button } from '../Button';
import DiscordLogo from '../Icons/Discord';

type DiscordCTAProps = {
  size?: 'default' | 'small';
};

export default function DiscordCTA({ size = 'default' }: DiscordCTAProps) {
  return (
    <div className="m-auto max-w-[70ch] border-t-[2px] border-slate-800 pt-16 text-indigo-500">
      <DiscordLogo size={32} />
      <h2 className="mt-6 text-xl font-medium text-white">Help shape the future of Inngest</h2>
      <p className="mb-6 mt-2 text-sm text-slate-400">
        Ask questions, give feedback, and share feature requests
      </p>
      <Button variant="secondary" href={process.env.NEXT_PUBLIC_DISCORD_URL} arrow="right">
        Join our Discord!
      </Button>
    </div>
  );
}
