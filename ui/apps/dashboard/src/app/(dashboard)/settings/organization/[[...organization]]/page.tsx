import { OrganizationProfile } from '@clerk/nextjs';
import colors from 'tailwindcss/colors';

export default function OrganizationSettingsPage() {
  return (
    <div className="min-h-0 flex-1">
      <OrganizationProfile
        routing="path"
        path="/settings/organization"
        appearance={{
          variables: {
            colorAlphaShade: colors.white, // white to hide the Clerk's scrollbar
          },
          elements: {
            rootBox: 'h-full',
            card: 'divide-x divide-slate-100',
            navbar: 'p-8 border-none',
            scrollBox: 'bg-white',
            pageScrollBox: '[scrollbar-width:none]', // hides the Clerk's scrollbar in Firefox
          },
        }}
      />
    </div>
  );
}
