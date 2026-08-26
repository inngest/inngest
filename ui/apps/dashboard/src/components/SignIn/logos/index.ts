import type { FC, SVGProps } from 'react';

export type CustomerLogo = {
  /** Company name. Used as the accessible label for the logo. */
  name: string;
  Logo: FC<SVGProps<SVGSVGElement>>;
};

/**
 * Customer logos for the sign-up trust panel, in render order.
 *
 * Empty until the cleared SVGs land in this directory -- `LogoWall` renders
 * nothing while it is, so the sign-up page stays correct in the meantime.
 *
 * To add one: drop `<name>.svg` here per README.md, then
 *
 *   import Replit from './replit.svg?react';
 *
 * and append `{ name: 'Replit', Logo: Replit }` below.
 */
export const customerLogos: CustomerLogo[] = [];
