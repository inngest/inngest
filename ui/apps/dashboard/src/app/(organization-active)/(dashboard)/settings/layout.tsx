import Layout from '@/components/Layout/Layout';
import { SettingsHeader } from '@/components/Settings/Header';
import Toaster from '@/components/Toaster';

type SettingsLayoutProps = {
  children: React.ReactNode;
};

export default async function SettingsLayout({ children }: SettingsLayoutProps) {
  return (
    <Layout>
      <div className="flex-col">
        <SettingsHeader />
        <div className="no-scrollbar overflow-y-scroll px-6">{children}</div>
        <Toaster />
      </div>
    </Layout>
  );
}
