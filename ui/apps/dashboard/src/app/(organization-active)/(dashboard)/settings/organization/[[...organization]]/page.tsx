import { OrganizationProfile } from '@clerk/nextjs';

export default async function OrganizationSettingsPage() {
  return (
    <div className="flex w-full flex-col justify-start">
      <OrganizationProfile
        routing="path"
        path="/settings/organization"
        appearance={{
          layout: {
            logoPlacement: 'none',
          },
          elements: {
            navbar: 'hidden',
            scrollBox: 'bg-canvasBase shadow-none',
            pageScrollBox: 'pt-6 px-2 w-full',
          },
        }}
      />
    </div>
  );
}
