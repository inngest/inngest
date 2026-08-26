import type { FC, SVGProps } from 'react';

import Avoca from './avoca.svg?react';
import Cohere from './cohere.svg?react';
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
 * To add one: drop `<name>.svg` here per README.md, import it with `?react`,
 * and append an entry below.
 *
 * TODO: Cubic is in the approved set but its asset has not landed yet. The
 * wall reflows, so it can be appended whenever it arrives.
 */
export const customerLogos: CustomerLogo[] = [
  { name: 'Replit', Logo: Replit, height: 22 },
  { name: 'ElevenLabs', Logo: ElevenLabs, height: 18 },
  { name: 'Cohere', Logo: Cohere, height: 20 },
  { name: 'SoundCloud', Logo: SoundCloud, height: 16 },
  { name: 'GitBook', Logo: GitBook, height: 22 },
  { name: 'Resend', Logo: Resend, height: 20 },
  { name: 'Avoca', Logo: Avoca, height: 22 },
];
