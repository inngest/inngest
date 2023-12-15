import { IconCloudflarePages } from '@inngest/components/icons/platforms/CloudflarePages';
import { IconRailway } from '@inngest/components/icons/platforms/Railway';
import { IconRender } from '@inngest/components/icons/platforms/Render';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';

export const platforms = ['cloudflare-pages', 'railway', 'render', 'vercel'] as const;
type Platform = (typeof platforms)[number];
function isPlatform(platform: string): platform is Platform {
  return platforms.includes(platform as Platform);
}

const platformInfo = {
  'cloudflare-pages': {
    Icon: IconCloudflarePages,
    text: 'Cloudflare Pages',
  },
  railway: {
    Icon: IconRailway,
    text: 'Railway',
  },
  render: {
    Icon: IconRender,
    text: 'Render',
  },
  vercel: {
    Icon: IconVercel,
    text: 'Vercel',
  },
} as const satisfies { [key in Platform]: { Icon: React.ComponentType; text: string } };

type Props = {
  platform: string | null | undefined;
};

export function PlatformInfo({ platform }: Props) {
  if (!platform) {
    return '-';
  }

  let Icon = null;
  let text = platform;
  if (isPlatform(platform)) {
    const info = platformInfo[platform];
    Icon = info.Icon;
    text = info.text;
  }

  return (
    <span className="flex">
      {Icon && <Icon className="mr-1" size={20} />}
      <span>{text}</span>
    </span>
  );
}
