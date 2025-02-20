import { Link } from '@inngest/components/Link/Link';

export default function Page() {
  return (
    <div>
      <h2 className="mb-2 text-xl">Choose language and framework</h2>
      <p className="text-subtle text-sm">
        We support <strong>all frameworks</strong> and languages. Below you will find a list of
        framework-specific bindings, as well as instructions on adding bindings to{' '}
        <Link
          href={
            'https://www.inngest.com/docs/learn/serving-inngest-functions#custom-frameworks?ref=dev-apps-choose-framework'
          }
          className="inline"
        >
          custom platforms
        </Link>
        . Learn more about serving inngest functions{' '}
        <Link
          href={
            'https://www.inngest.com/docs/learn/serving-inngest-functions?ref=dev-apps-choose-framework'
          }
          className="inline"
        >
          here
        </Link>
        .
      </p>
    </div>
  );
}
