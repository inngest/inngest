import AppNavigation from '@/components/Navigation/AppNavigation';
import Toaster from '@/components/Toaster';
import { Contexts } from './Contexts';

type RootLayoutProps = {
  params: {
    environmentSlug: string;
  };
  children: React.ReactNode;
};

export default function RootLayout({ params, children }: RootLayoutProps) {
  const environmentSlug = params.environmentSlug ?? 'production';
  return (
    <Contexts envSlug={environmentSlug}>
      <div className="isolate flex h-full flex-col">
        <AppNavigation environmentSlug={environmentSlug} />
        {children}
      </div>
      <Toaster />
    </Contexts>
  );
}
