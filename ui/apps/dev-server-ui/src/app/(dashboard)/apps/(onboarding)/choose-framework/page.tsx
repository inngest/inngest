import { Link } from '@inngest/components/Link/Link';

import FrameworkList from '@/components/Onboarding/FrameworkList';
import frameworksData from '@/components/Onboarding/frameworks.json';

export default function Page() {
  return (
    <FrameworkList
      frameworksData={frameworksData}
      title="Choose language and framework"
      description={
        <>
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
        </>
      }
    />
  );
}
