import { Button } from '@inngest/components/Button';
import AvatarMenu from '../../Layout/AvatarMenu';
import { OrgMenu } from '../../Layout/OrgMenu';
import OrgButton from '../../Layout/OrgButton';
import SearchTrigger from '../../Layout/SearchTrigger';
import type { VerificationActions, VerificationState } from '../types';

export function DemoHeader({ state }: { state: VerificationState }) {
  const account = state.selectedAccount;
  const profile = {
    orgName: account.companyName,
    displayName: account.displayName,
    isMarketplace: false,
    marketplace: null,
    orgProfilePic: null,
    userProfilePic: null,
  };
  return (
    <header className="flex min-h-[42px] shrink-0 flex-wrap items-center justify-between gap-2 px-3 py-1">
      <OrgMenu profile={profile} showOnboardingWidget={() => undefined}>
        <OrgButton profile={profile} />
      </OrgMenu>
      <div className="flex items-center gap-3">
        <SearchTrigger envSlug="production" envName="Production" />
        <AvatarMenu profile={profile} />
      </div>
    </header>
  );
}

export function DemoFooter({
  onContactSupport,
}: {
  onContactSupport: VerificationActions['onContactSupport'];
}) {
  return (
    <footer className="text-muted flex min-h-[30px] flex-wrap items-center gap-1 px-2 text-xs">
      <Button
        kind="secondary"
        appearance="ghost"
        size="small"
        label="Support"
        onClick={onContactSupport}
      />
      <span aria-hidden="true">|</span>
      <a href="https://www.inngest.com/docs" className="px-2 hover:underline">
        Docs
      </a>
      <span aria-hidden="true">|</span>
      <a
        href="https://www.inngest.com/changelog"
        className="px-2 hover:underline"
      >
        Changelog
      </a>
      <span aria-hidden="true">|</span>
      <a href="https://status.inngest.com" className="px-2 hover:underline">
        Status
      </a>
    </footer>
  );
}
