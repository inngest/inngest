import { OrganizationProfile } from '@clerk/tanstack-react-start';
import { createFileRoute, useLocation } from '@tanstack/react-router';

export const Route = createFileRoute('/_authed/settings/organization/$')({
  component: OrganizationSettingsPage,
});

function OrganizationSettingsPage() {
  const location = useLocation();

  return (
    <div className="flex w-full flex-col justify-start">
      <OrganizationProfile
        key={location.pathname}
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
