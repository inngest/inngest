import { CodeKey } from '@inngest/components/CodeKey';

import { Alert } from '@/components/Alert';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import { InlineCode } from './InlineCode';

const Query = graphql(`
  query GetSigningKey($environmentID: ID!) {
    environment: workspace(id: $environmentID) {
      webhookSigningKey
    }
  }
`);

type Props = {
  params: {
    environmentSlug: string;
  };
};

export default async function Page({ params }: Props) {
  const environment = await getEnvironment({
    environmentSlug: params.environmentSlug,
  });

  const res = await graphqlAPI.request(Query, {
    environmentID: environment.id,
  });

  const maskedSigningKey = res.environment.webhookSigningKey.replace(
    /signkey-(prod|test)-(.{0,4}).+/,
    'signkey-$1-$2...'
  );

  return (
    <div className="flex place-content-center">
      <div className="mt-8 max-w-3xl rounded-md border border-slate-200 px-8 pb-8 pt-6">
        <h2 className="mb-6 text-lg font-semibold text-slate-800">Inngest Signing Key</h2>

        <CodeKey
          fullKey={res.environment.webhookSigningKey}
          maskedKey={maskedSigningKey}
          label="Signing key"
        />

        <p className="mt-8 text-sm leading-relaxed text-slate-700">
          Use this <span className="font-bold text-slate-800">secret signing key</span> with the
          Inngest SDK to enable Inngest to securely communicate with your application.
        </p>

        <p className="mt-4 text-sm leading-relaxed text-slate-700">
          You will need to set this signing key as the <InlineCode value="INNGEST_SIGNING_KEY" />{' '}
          environment variable in your hosting provider. Alternatively, you can explicitly pass the
          signing key in the {"SDK's"} serve {"handler's"} options argument. Read the{' '}
          <InlineCode value="serve" />{' '}
          <a
            className="font-semibold text-indigo-500 hover:text-indigo-700 hover:underline"
            href={
              'https://www.inngest.com/docs/sdk/serve?ref=app-manage-signing-key#signing-key' as any
            }
            target="_blank"
          >
            reference
          </a>{' '}
          for more information.
        </p>

        <Alert severity="info" className="mt-8">
          If {"you're"} using the{' '}
          <a
            className="font-semibold text-indigo-600 transition-all hover:text-indigo-800 hover:underline"
            href="https://vercel.com/integrations/inngest"
            target="_blank"
          >
            Vercel integration
          </a>
          , the signing key will be set automatically for any project that you enable.
        </Alert>
      </div>
    </div>
  );
}
