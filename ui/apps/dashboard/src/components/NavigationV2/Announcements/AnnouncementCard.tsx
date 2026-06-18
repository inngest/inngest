import { Button } from '@inngest/components/Button';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip/Tooltip';
import { RiSubtractLine } from '@remixicon/react';

import { isSafeCTAURL } from '@/components/ActiveBanners/safeUrl';
import type { Announcement } from './announcements';

// Every card renders at this fixed inner height so the stack and its flip/
// dismiss transitions never jump between cards with vs. without an image.
// Kept as a literal class string (not interpolated) so Tailwind can see it.
export const CARD_CONTENT_HEIGHT_CLASS = 'h-[194px]';

// Fixed height for the top illustration slot. Cards without an image leave it
// blank so the title/body still line up vertically across cards in the stack.
const CARD_IMAGE_HEIGHT_CLASS = 'h-[110px]';

const CARD_CLASS =
  'group text-basis bg-canvasSubtle border-subtle block overflow-hidden rounded border leading-tight shadow-md';

export default function AnnouncementCard({
  announcement,
  onDismiss,
  isFront = true,
}: {
  announcement: Announcement;
  onDismiss?: () => void;
  /** Background (peeking) cards drop their interactive controls — they're occluded. */
  isFront?: boolean;
}) {
  const { title, body, imageUrl, imageUrlDark, cta } = announcement;
  const ctaIsSafe = cta ? isSafeCTAURL(cta.href) : false;
  const ctaIsExternal = cta ? /^https?:/i.test(cta.href) : false;

  const content = (
    <div className={`flex ${CARD_CONTENT_HEIGHT_CLASS} flex-col`}>
      <div className="relative">
        <div
          className={`bg-canvasBase ${CARD_IMAGE_HEIGHT_CLASS} w-full overflow-hidden`}
        >
          {imageUrl && (
            <img
              src={imageUrl}
              alt=""
              className={`h-full w-full object-cover ${
                imageUrlDark ? 'dark:hidden' : ''
              }`}
              draggable={false}
            />
          )}
          {imageUrlDark && (
            <img
              src={imageUrlDark}
              alt=""
              className="hidden h-full w-full object-cover dark:block"
              draggable={false}
            />
          )}
        </div>
        {isFront && onDismiss && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                icon={<RiSubtractLine className="text-light" />}
                kind="secondary"
                appearance="ghost"
                size="small"
                className="absolute right-1 top-1 shrink-0 bg-transparent opacity-0 transition-opacity hover:bg-transparent group-hover:opacity-100"
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  onDismiss();
                }}
              />
            </TooltipTrigger>
            <TooltipContent side="right" className="max-w-40">
              <p>Dismiss</p>
            </TooltipContent>
          </Tooltip>
        )}
      </div>

      <div className="px-2 pb-6">
        <p className="mt-3 truncate text-base">{title}</p>

        <p className="text-muted mt-0.5 line-clamp-2 text-[13px]">{body}</p>
      </div>
    </div>
  );

  // The whole front card acts as the CTA link when one is present and safe.
  if (isFront && cta && ctaIsSafe) {
    return (
      <a
        href={cta.href}
        target={ctaIsExternal ? '_blank' : undefined}
        rel={ctaIsExternal ? 'noopener noreferrer' : undefined}
        className={`${CARD_CLASS} hover:border-muted transition-colors`}
      >
        {content}
      </a>
    );
  }

  return <div className={CARD_CLASS}>{content}</div>;
}
