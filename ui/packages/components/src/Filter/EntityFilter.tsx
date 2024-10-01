import { useRef, useState } from 'react';

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
  const [temporarySelectedValues, setTemporarySelectedValues] = useState(selectedEntities);
  const comboboxRef = useRef<HTMLButtonElement>(null);

  const selectedValues = entities.filter((entity) =>
    temporarySelectedValues.some((id) => id === entity.id)
  );
  const areAllEntitiesSelected = temporarySelectedValues.length === entities.length;

  const filteredOptions =
    query === ''
      ? entities
      : entities.filter((entity) => {
          return entity.name.toLowerCase().includes(query.toLowerCase());
        });

  const isSelectionChanged = () => {
    if (temporarySelectedValues.length !== selectedEntities.length) {
      return true;
    }
    const tempSet = new Set(temporarySelectedValues);
    return selectedEntities.some((id) => !tempSet.has(id));
  };

  const isDisabledApply = !isSelectionChanged();
  const isDisabledReset = temporarySelectedValues.length === 0 && selectedEntities.length === 0; // Disable if no items are selected

  const handleApply = () => {
    onFilterChange(temporarySelectedValues);
    // Close the Select dropdown
    if (comboboxRef.current) {
      comboboxRef.current.click();
    }
  };

  const handleReset = () => {
    setTemporarySelectedValues([]);
  };

  return (
    <SelectWithSearch
      multiple
      value={selectedValues}
      onChange={(value: Option[]) => {
        const newValue: string[] = [];
        value.forEach((option) => {
          newValue.push(option.id);
        });
        setTemporarySelectedValues(newValue);
      }}
      label={type}
      isLabelVisible
    >
      <SelectWithSearch.Button isLabelVisible className={className} ref={comboboxRef}>
        <div className="min-w-7 max-w-24 truncate text-nowrap text-left">
          {temporarySelectedValues.length === 1 && !areAllEntitiesSelected && (
            <span>{selectedValues[0]?.name}</span>
          )}
          {temporarySelectedValues.length > 1 && !areAllEntitiesSelected && (
            <span>
              {temporarySelectedValues.length} {type}s
            </span>
          )}
          {(temporarySelectedValues.length === 0 || areAllEntitiesSelected) && <span>All</span>}
        </div>
      </SelectWithSearch.Button>
      <SelectWithSearch.Options>
        <SelectWithSearch.SearchInput
          displayValue={(option: Option) => option?.name}
          placeholder={`Search for ${type}`}
          onChange={(event) => setQuery(event.target.value)}
        />
        <div className="max-h-64 overflow-scroll">
          {filteredOptions.map((option) => (
            <SelectWithSearch.CheckboxOption key={option.id} option={option}>
              {option.name}
            </SelectWithSearch.CheckboxOption>
          ))}
        </div>
        <SelectWithSearch.Footer
          onReset={handleReset}
          onApply={handleApply}
          disabledReset={isDisabledReset}
          disabledApply={isDisabledApply}
        />
      </SelectWithSearch.Options>
    </SelectWithSearch>
  );
}
