import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import NewPage from './page-new';
import OldPage from './page-old';

type Props = {
  params: {
    environmentSlug: string;
  };
};

export default async function Page({ params }: Props) {
  const isRotationEnabled = await getBooleanFlag('signing-key-rotation');
  if (!isRotationEnabled) {
    return <OldPage environmentSlug={params.environmentSlug} />;
  }

  return <NewPage />;
}
