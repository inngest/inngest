import { OrganizationProfile } from '@clerk/nextjs';

export default function OrganizationSettingsPage() {
  return (
    <div className="min-h-0 flex-1">
      <OrganizationProfile
        routing="path"
        path="/settings/organization"
        appearance={{
          elements: {
            rootBox: 'h-full',
            card: 'divide-x divide-slate-100',
            navbar: 'p-8 border-none',
            scrollBox: 'bg-white',
            pageScrollBox: '[scrollbar-width:none]', // hides the Clerk's scrollbar
            profileSection__organizationDanger: 'hidden', // hides the "Danger Zone" section, i.e. leaving orgs
          },
        }}
      />
    </div>
  );
}
