import { useMemo } from 'react';

import {
  CodeElement,
  ElementWrapper,
  IDElement,
  LinkElement,
  TextElement,
} from '../DetailsCard/Element';
import { type RunDeferEntry } from '../SharedContext/useGetRun';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { StatusCell } from '../Table/Cell';
import { CollapsibleSection } from './CollapsibleSection';

type Props = {
  defers: RunDeferEntry[] | undefined;
};

export const DefersSection = ({ defers }: Props) => {
  if (!defers || defers.length === 0) {
    return null;
  }

  return (
    <CollapsibleSection title={`Defers (${defers.length})`}>
      <div className="flex flex-col gap-4">
        {defers.map((defer) => (
          <DeferRow key={defer.id} defer={defer} />
        ))}
      </div>
    </CollapsibleSection>
  );
};

const DeferRow = ({ defer }: { defer: RunDeferEntry }) => {
  const { pathCreator } = usePathCreator();

  return (
    <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4">
      <ElementWrapper label="Defer ID">
        <IDElement>{defer.id}</IDElement>
      </ElementWrapper>

      <ElementWrapper label="Function">
        <LinkElement href={pathCreator.function({ functionSlug: defer.fnSlug })}>
          {defer.fnSlug}
        </LinkElement>
      </ElementWrapper>

      <ElementWrapper label="Status">
        {defer.run ? (
          <StatusCell status={defer.run.status} />
        ) : (
          <TextElement>{defer.status.toLowerCase()}</TextElement>
        )}
      </ElementWrapper>

      <ElementWrapper label="Run">
        {defer.run ? (
          <LinkElement href={pathCreator.runPopout({ runID: defer.run.id })}>
            {defer.run.id}
          </LinkElement>
        ) : (
          <TextElement>—</TextElement>
        )}
      </ElementWrapper>

      <ElementWrapper label="Input">
        <InputCell input={defer.input} />
      </ElementWrapper>
    </div>
  );
};

const InputCell = ({ input }: { input: unknown }) => {
  const formatted = useMemo(() => {
    if (input == null || input === '') return null;
    const raw = typeof input === 'string' ? input : JSON.stringify(input);
    try {
      return JSON.stringify(JSON.parse(raw));
    } catch {
      return raw;
    }
  }, [input]);

  if (formatted === null) {
    return <TextElement>—</TextElement>;
  }
  return <CodeElement value={formatted} />;
};
