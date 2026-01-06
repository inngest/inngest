import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';

export const FunctionInfo = () => (
  <Info
    text="List of all Inngest functions in the current environment."
    action={
      <Link href={'https://www.inngest.com/docs/functions'} target="_blank">
        Learn how to create a function
      </Link>
    }
  />
);
