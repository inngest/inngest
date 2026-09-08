import { cn } from '@inngest/components/utils/classNames';

import { customerLogos } from './logos';

/**
 * Renders on the gray panel on desktop and on `canvasBase` in the left column
 * on mobile. Both are dark, because the auth routes are pinned to a dark
 * theme, so the white-mode exports are correct at every breakpoint.
 *
 * Heights come from the registry rather than a shared class: the wordmarks
 * differ too much in aspect ratio for one height to look balanced.
 */
export default function LogoWall({ className }: { className?: string }) {
  if (customerLogos.length === 0) {
    return null;
  }

  return (
    <ul
      className={cn(
        'flex flex-wrap items-center justify-center gap-x-8 gap-y-5',
        className,
      )}
    >
      {customerLogos.map(({ name, Logo, height }) => (
        <li key={name} className="flex items-center">
          <Logo
            role="img"
            aria-label={name}
            height={height}
            className="w-auto opacity-70 transition-opacity hover:opacity-100"
          />
        </li>
      ))}
    </ul>
  );
}
