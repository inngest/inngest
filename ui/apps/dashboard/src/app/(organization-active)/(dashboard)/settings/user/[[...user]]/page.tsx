import { UserProfile } from '@clerk/nextjs';

export default async function UserSettingsPage() {
  return (
    <UserProfile
      routing="path"
      path="/settings/user"
      appearance={{
        layout: {
          logoPlacement: 'none',
        },
        elements: {
          rootBox: 'h-full',
          card: '',
          navbar: 'hidden',
          scrollBox: 'bg-white',
          pageScrollBox: '[scrollbar-width:none]', // hides the Clerk's scrollbar
        },
      }}
    />
  );
}
