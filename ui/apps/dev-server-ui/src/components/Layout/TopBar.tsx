import { InngestLogo } from '@inngest/components/icons/logos/InngestLogo';
import { Link } from '@tanstack/react-router';

import { SettingsMenu } from '../NavigationV2/SettingsMenu';

export default function TopBar() {
  return (
    <header className="bg-canvasSubtle relative z-[60] flex h-[48px] shrink-0 items-center justify-between gap-3 px-3">
      <div className="flex h-8 items-center gap-1.5">
        <Link to="/">
          <InngestLogo className="text-basis" width={96} />
        </Link>
        <span className="text-primary-intense text-[11px] font-medium leading-none">
          DEVELOPMENT SERVER
        </span>
      </div>
      <div className="flex items-center gap-3">
        <SettingsMenu />
      </div>
    </header>
  );
}
