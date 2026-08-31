import type { FC, SVGProps } from 'react';

import Avoca from './avoca.svg?react';
import Cohere from './cohere.svg?react';
import Cubic from './cubic.svg?react';
import ElevenLabs from './elevenlabs.svg?react';
import GitBook from './gitbook.svg?react';
import Replit from './replit.svg?react';
import Resend from './resend.svg?react';
import SoundCloud from './soundcloud.svg?react';

export type CustomerLogo = {
  /** Company name. Used as the accessible label for the logo. */
  name: string;
  Logo: FC<SVGProps<SVGSVGElement>>;
  /**
   * Rendered height in pixels. Set per logo rather than uniformly: these
   * wordmarks range from 4:1 to 9.4:1, so a single height would make the
   * widest read as nearly twice the size of the narrowest. These values
   * balance them by optical weight instead.
   */
  height: number;
};

/**
 * Customer logos for the sign-up trust panel, in render order.
 *
 * To add one: drop `<name>.svg` here, import it with `?react`, and append an
 * entry below. Two things to fix in a Figma export first, because both fail
 * silently rather than visibly:
 *
 * 1. Strip `width`/`height` from the root `<svg>`, keeping only `viewBox`. A
 *    leftover `width` beats the aspect ratio against the height set here and
 *    renders the logo horizontally squashed.
 * 2. If the export paints one solid path through a `mask-type:alpha` mask
 *    built from the letterforms, flatten it to those letterforms. The mask
 *    stops applying once inlined as JSX and the solid path renders as a bar.
 *    See `soundcloud.svg`, which was flattened for this reason.
 *
 * Keep fills white rather than `currentColor`: every surface behind these is
 * dark, and white inside a `<mask>` means "reveal", so a themed colour there
 * would hide the artwork instead of colouring it.
 *
 * All eight of the approved set are present.
 */
export const customerLogos: CustomerLogo[] = [
  { name: 'Replit', Logo: Replit, height: 22 },
  { name: 'Cubic', Logo: Cubic, height: 20 },
  { name: 'ElevenLabs', Logo: ElevenLabs, height: 18 },
  { name: 'Cohere', Logo: Cohere, height: 20 },
  { name: 'SoundCloud', Logo: SoundCloud, height: 16 },
  { name: 'GitBook', Logo: GitBook, height: 22 },
  { name: 'Resend', Logo: Resend, height: 20 },
  { name: 'Avoca', Logo: Avoca, height: 22 },
];
