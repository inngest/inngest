import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import NewLayout from './newLayout';
import OldLayout from './oldLayout';

type Props = React.PropsWithChildren<{
  params: {
    externalID: string;
  };
}>;

export default async function Layout({ children, params: { externalID } }: Props) {
  const newIANav = await getBooleanFlag('new-ia-nav');

  return newIANav ? (
    <NewLayout
      children={children}
      params={{
        externalID,
      }}
    />
  ) : (
    <OldLayout
      children={children}
      params={{
        externalID,
      }}
    />
  );
}
