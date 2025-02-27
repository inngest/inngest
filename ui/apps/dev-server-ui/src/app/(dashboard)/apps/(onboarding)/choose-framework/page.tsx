import NextLink from 'next/link';
import { Link } from '@inngest/components/Link/Link';
import { Pill } from '@inngest/components/Pill/Pill';

import frameworksData from './frameworks.json';

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

      <ul className="mt-8 flex flex-col gap-4">
        {frameworksData.frameworks.map((framework) => (
          <li key={framework.name} className="border-subtle rounded-sm border">
            <NextLink
              href={framework.url}
              target="_blank"
              className="hover:bg-canvasSubtle flex items-center justify-between p-3"
            >
              <div className="flex items-center">
                <div className="bg-canvasMuted mr-3 h-12 w-12 rounded-sm">{framework.logo}</div>
                <p className="mr-1">{framework.name}</p>
                {framework.sdk_version_supported && <Pill>{framework.sdk_version_supported}</Pill>}
              </div>
              <Pill
                appearance="outlined"
                kind={framework.sdk_language.toLowerCase() === 'typescript' ? 'primary' : 'default'}
              >
                {framework.sdk_language}
              </Pill>
            </NextLink>
          </li>
        ))}
      </ul>
    </div>
  );
}
