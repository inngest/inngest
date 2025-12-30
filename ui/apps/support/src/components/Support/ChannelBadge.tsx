import {
  RiSlackLine,
  RiDiscordLine,
  RiMailLine,
  RiMessage2Line,
} from "@remixicon/react";
import type { TicketChannel } from "@/data/plain";

type ChannelBadgeProps = {
  channel?: TicketChannel;
};

const channelConfig: Record<
  TicketChannel,
  { icon: React.ComponentType<{ className?: string }>; label: string }
> = {
  SLACK: { icon: RiSlackLine, label: "Slack" },
  DISCORD: { icon: RiDiscordLine, label: "Discord" },
  API: { icon: RiMessage2Line, label: "Portal" },
  EMAIL: { icon: RiMailLine, label: "Email" },
};

export function ChannelBadge({ channel }: ChannelBadgeProps) {
  if (!channel) return null;
  const config = channelConfig[channel];

  if (!config) return null;

  const Icon = config.icon;

  return (
    <div className="flex items-center gap-2">
      <Icon className="text-muted h-4 w-4" />
      <div className="text-muted flex flex-col justify-center leading-none hidden md:block">
        <p className="leading-4">{config.label}</p>
      </div>
    </div>
  );
}
