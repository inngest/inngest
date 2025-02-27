import NextLink from 'next/link';
import { Pill } from '@inngest/components/Pill/Pill';

import templatesData from './templates.json';

export default function Page() {
  return (
    <div>
      <h2 className="mb-2 text-xl">Choose a template</h2>
      <p className="text-subtle text-sm">
        Using a template provides quick setup and integration of Inngest into your project. It
        demonstrates key functionality, allowing you to send and receive events with minimal
        configuration..
      </p>

      <ul className="mt-8 flex flex-col gap-4">
        {templatesData.templates.map((templates) => (
          <li key={templates.name} className="border-subtle rounded-sm border">
            <NextLink
              href={templates.url}
              target="_blank"
              className="hover:bg-canvasSubtle flex items-center justify-between p-3"
            >
              <div className="flex items-center">
                <div className="bg-canvasMuted mr-3 h-12 w-12 rounded-sm">{templates.logo}</div>
                <p>{templates.name}</p>
              </div>
              <Pill
                appearance="outlined"
                kind={templates.sdk_language.toLowerCase() === 'typescript' ? 'primary' : 'default'}
              >
                {templates.sdk_language}
              </Pill>
            </NextLink>
          </li>
        ))}
      </ul>
    </div>
  );
}
