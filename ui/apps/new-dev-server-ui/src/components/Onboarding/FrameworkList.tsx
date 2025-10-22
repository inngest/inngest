'use client'

import { useMemo, useState } from 'react'
import { Search } from '@inngest/components/Forms/Search'
import { ThemeImage } from '@inngest/components/Image/Image'
import { Pill } from '@inngest/components/Pill/NewPill'
import { Select, type Option } from '@inngest/components/Select/Select'

import { useTracking } from '@/hooks/useTracking'
import { Link } from '@tanstack/react-router'

type Framework = {
  framework: string
  logo: {
    dark: string
    light: string
  }
  version_supported?: string
  link: {
    url: string
  }
  language: string
}

type FrameworkListProps = {
  frameworksData: Framework[]
  title: React.ReactNode
  description: React.ReactNode
}

function getPillAppearance(language: string) {
  if (language.toLowerCase() === 'typescript') {
    return 'primary'
  } else if (language.toLowerCase() === 'python') {
    return 'info'
  } else {
    return 'default'
  }
}

export default function FrameworkList({
  frameworksData,
  title,
  description,
}: FrameworkListProps) {
  const { trackEvent } = useTracking()
  // Extract unique languages from frameworks data
  const languageOptions = useMemo(() => {
    const uniqueLanguages = Array.from(
      new Set(frameworksData.map((framework) => framework.language)),
    )

    return uniqueLanguages.map((language) => ({
      id: language.toLowerCase(),
      name: language,
    }))
  }, [])

  const [selectedValues, setSelectedValues] = useState<Option[]>([])
  const [searchQuery, setSearchQuery] = useState('')

  const filteredFrameworks = useMemo(() => {
    // First filter by language selection
    let filtered = frameworksData

    if (selectedValues.length > 0) {
      filtered = filtered.filter((framework) =>
        selectedValues.some(
          (option) => option.id === framework.language.toLowerCase(),
        ),
      )
    }

    // Then filter by search query
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter((framework) =>
        framework.framework.toLowerCase().includes(query),
      )
    }

    return filtered
  }, [frameworksData, selectedValues, searchQuery])

  return (
    <div>
      <h2 className="mb-2 text-xl">{title}</h2>
      <p className="text-subtle text-sm">{description}</p>
      <div className="mb-4 mt-6 flex items-center justify-between ">
        <Search
          name="search"
          placeholder="Search framework name"
          value={searchQuery}
          onUpdate={(value) => setSearchQuery(value)}
          className="w-80"
        />
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
              {selectedValues.length === 0 ||
              selectedValues.length === languageOptions.length ? (
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
                  <label className="text-sm first-letter:capitalize">
                    {option.name}
                  </label>
                </span>
              </Select.CheckboxOption>
            ))}
          </Select.Options>
        </Select>
      </div>

      <ul className="flex flex-col gap-4">
        {filteredFrameworks.length > 0 ? (
          filteredFrameworks.map((framework) => (
            <li
              key={framework.framework}
              className="border-subtle rounded-sm border"
            >
              <Link
                onClick={() =>
                  trackEvent('cli/onboarding.action', {
                    type: 'btn-click',
                    label: 'choose-framework-from-list',
                    framework: framework.framework,
                  })
                }
                to={framework.link.url}
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
                  {framework.version_supported && (
                    <Pill>{framework.version_supported}</Pill>
                  )}
                </div>

                <Pill
                  appearance="outlined"
                  kind={getPillAppearance(framework.language)}
                >
                  {framework.language}
                </Pill>
              </Link>
            </li>
          ))
        ) : (
          <li className="text-muted py-4 text-center text-sm">
            No frameworks match your search criteria
          </li>
        )}
      </ul>
    </div>
  )
}
