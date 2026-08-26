import { cn } from '@inngest/components/utils/classNames';

import { customerLogos } from './logos';

/**
 * Renders on the dark mesh gradient on desktop and on `canvasBase` in the
 * left column on mobile, so the logos are recolored via `currentColor`
 * rather than shipping a light and a dark copy of each.
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
      {customerLogos.map(({ name, Logo }) => (
        <li key={name} className="flex items-center">
          <Logo
            role="img"
            aria-label={name}
            className="h-5 w-auto opacity-80 transition-opacity hover:opacity-100"
          />
        </li>
      ))}
    </ul>
  );
}
