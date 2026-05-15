import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';

export const SigningKeyInfo = () => (
  <Info
    text="Use the secret signing key with the Inngest SDK to enable us securely communicate with your
        application."
    action={
      <Link
        href={'https://www.inngest.com/docs/platform/signing-keys'}
        target="_blank"
      >
        Learn how create a webhook
      </Link>
    }
  />
);
