import { OrganizationProfile } from '@clerk/nextjs';

export default async function OrganizationSettingsPage() {
  return (
    <div className="min-h-0 flex-1">
      <OrganizationProfile
        routing="path"
        path="/settings/organization"
        appearance={{
          elements: {
            rootBox: 'h-full',
            navbar: 'hidden',
            scrollBox: 'bg-white',
            pageScrollBox: '[scrollbar-width:none]', // hides the Clerk's scrollbar
          },
        }}
      />
    </div>
  );
}
