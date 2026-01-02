import {
  RiDiscordLine,
  RiMailLine,
  RiMessage2Line,
  RiSlackLine,
} from "@remixicon/react";
import type { TicketChannel } from "@/data/plain";

type ChannelBadgeProps = {
  channel?: TicketChannel;
  showLabel?: boolean;
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

export function ChannelBadge({ channel, showLabel }: ChannelBadgeProps) {
  if (!channel) return null;
  const config = channelConfig[channel];

  const Icon = config.icon;
  // If showLabel is explicitly set, use it. Otherwise, default to showing on desktop only (original behavior)
  const shouldShowLabel = showLabel !== undefined ? showLabel : undefined; // undefined means use responsive classes

  return (
    <div className="flex items-center gap-1">
      <Icon className="text-muted h-4 w-4" />
      {shouldShowLabel === true ? (
        <span className="text-muted text-xs leading-4">{config.label}</span>
      ) : shouldShowLabel === false ? null : (
        <div className="text-muted flex flex-col justify-center leading-none hidden md:block">
          <p className="leading-4">{config.label}</p>
        </div>
      )}
    </div>
  );
}
