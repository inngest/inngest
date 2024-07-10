import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import RootLayout from '@/components/Layout/Root';
import OldRootLayout from '@/components/Layout/old/Root';

type ParentLayoutProps = {
  params: {
    environmentSlug: string;
  };
  children: React.ReactNode;
};

export default async function ParentLayout({
  params: { environmentSlug },
  children,
}: ParentLayoutProps) {
  const newIANav = await getBooleanFlag(' ');

  return newIANav ? (
    <RootLayout environmentSlug={environmentSlug}>{children}</RootLayout>
  ) : (
    <OldRootLayout environmentSlug={environmentSlug}>{children}</OldRootLayout>
  );
}
