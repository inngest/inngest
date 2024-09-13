import { useState } from 'react';

import { SelectWithSearch, type Option } from '../Select/Select';

type EntityFilterProps = {
  type: 'app' | 'function';
  selectedEntities: string[];
  entities: Option[];
  onFilterChange: (value: string[]) => void;
  className?: string;
};

export default function EntityFilter({
  selectedEntities,
  entities,
  onFilterChange,
  type,
  className,
}: EntityFilterProps) {
  const [query, setQuery] = useState('');
  const selectedValues = entities.filter((entity) =>
    selectedEntities.some((id) => id === entity.id)
  );
  const areAllEntitiesSelected = selectedEntities.length === entities.length;

  const filteredOptions =
    query === ''
      ? entities
      : entities.filter((entity) => {
          return entity.name.toLowerCase().includes(query.toLowerCase());
        });

  return (
    <SelectWithSearch
      multiple
      value={selectedValues}
      onChange={(value: Option[]) => {
        const newValue: string[] = [];
        value.forEach((option) => {
          newValue.push(option.id);
        });
        onFilterChange(newValue);
      }}
      label={type}
      isLabelVisible
    >
      <SelectWithSearch.Button isLabelVisible className={className}>
        <div className="min-w-7 max-w-24 truncate text-nowrap text-left">
          {selectedEntities.length === 1 && !areAllEntitiesSelected && (
            <span>{selectedValues[0]?.name}</span>
          )}
          {selectedEntities.length > 1 && !areAllEntitiesSelected && (
            <span>
              {selectedEntities.length} {type}s
            </span>
          )}
          {(selectedEntities.length === 0 || areAllEntitiesSelected) && <span>All</span>}
        </div>
      </SelectWithSearch.Button>
      <SelectWithSearch.Options>
        <SelectWithSearch.SearchInput
          displayValue={(option: Option) => option?.name}
          placeholder={`Search for ${type}`}
          onChange={(event) => setQuery(event.target.value)}
        />
        {filteredOptions.map((option) => (
          <SelectWithSearch.CheckboxOption key={option.id} option={option}>
            {option.name}
          </SelectWithSearch.CheckboxOption>
        ))}
        <SelectWithSearch.Footer onReset={() => onFilterChange([])} />
      </SelectWithSearch.Options>
    </SelectWithSearch>
  );
}
