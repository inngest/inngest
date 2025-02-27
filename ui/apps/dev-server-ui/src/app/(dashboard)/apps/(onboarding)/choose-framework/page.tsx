'use client';

import { useMemo, useState } from 'react';
import NextLink from 'next/link';
import { ThemeImage } from '@inngest/components/Image/Image';
import { Link } from '@inngest/components/Link/Link';
import { Pill } from '@inngest/components/Pill/Pill';
import { Select, type Option } from '@inngest/components/Select/Select';

import frameworksData from './frameworks.json';

function getPillAppearance(language: string) {
  if (language.toLowerCase() === 'typescript') {
    return 'primary';
  } else if (language.toLowerCase() === 'python') {
    return 'info';
  } else {
    return 'default';
  }
}

export default function Page() {
  // Extract unique languages from frameworks data
  const languageOptions = useMemo(() => {
    const uniqueLanguages = Array.from(
      new Set(frameworksData.map((framework) => framework.language))
    );

    return uniqueLanguages.map((language) => ({
      id: language.toLowerCase(),
      name: language,
    }));
  }, []);

  const [selectedValues, setSelectedValues] = useState<Option[]>([]);

  const filteredFrameworks = useMemo(() => {
    if (selectedValues.length === 0) {
      return frameworksData; // Show all frameworks if no language is selected
    }

    return frameworksData.filter((framework) =>
      selectedValues.some((option) => option.id === framework.language.toLowerCase())
    );
  }, [selectedValues]);

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
      <div className="mb-4 mt-6 flex items-center justify-end ">
        <Select
          size="small"
          multiple
          value={selectedValues}
          onChange={(value: Option[]) => setSelectedValues(value)}
          label="Language"
          isLabelVisible
        >
          <Select.Button isLabelVisible>
            <div className="text-left">
              {selectedValues.length === 0 || selectedValues.length === languageOptions.length ? (
                <span>All</span>
              ) : selectedValues.length === 1 && selectedValues[0] ? (
                <span>{selectedValues[0].name}</span>
              ) : (
                <span>{selectedValues.length} selected</span>
              )}
            </div>
          </Select.Button>
          <Select.Options>
            {languageOptions.map((option) => (
              <Select.CheckboxOption key={option.id} option={option}>
                <span className="flex items-center gap-1 lowercase">
                  <label className="text-sm first-letter:capitalize">{option.name}</label>
                </span>
              </Select.CheckboxOption>
            ))}
          </Select.Options>
        </Select>
      </div>

      <ul className="flex flex-col gap-4">
        {filteredFrameworks.map((framework) => (
          <li key={framework.framework} className="border-subtle rounded-sm border">
            <NextLink
              href={framework.link.url}
              target="_blank"
              className="hover:bg-canvasSubtle flex items-center justify-between p-3"
            >
              <div className="flex items-center">
                <div className="bg-canvasMuted mr-3 flex h-12 w-12 items-center justify-center rounded-sm">
                  {framework.logo.light && framework.logo.dark && (
                    <ThemeImage
                      width={30}
                      height={30}
                      lightSrc={framework.logo.light}
                      darkSrc={framework.logo.dark}
                      alt={framework.framework + ' logo'}
                    />
                  )}
                </div>
                <p className="mr-1">{framework.framework}</p>
                {framework.version_supported && <Pill>{framework.version_supported}</Pill>}
              </div>

              <Pill appearance="outlined" kind={getPillAppearance(framework.language)}>
                {framework.language}
              </Pill>
            </NextLink>
          </li>
        ))}
      </ul>
    </div>
  );
}
