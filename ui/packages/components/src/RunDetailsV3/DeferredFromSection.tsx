import { ElementWrapper, LinkElement, TextElement } from '../DetailsCard/Element';
import { type RunDeferredFromEntry } from '../SharedContext/useGetRun';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { StatusCell } from '../Table/Cell';
import { CollapsibleSection } from './CollapsibleSection';

type Props = {
  deferredFrom: RunDeferredFromEntry | undefined | null;
};

export const DeferredFromSection = ({ deferredFrom }: Props) => {
  const { pathCreator } = usePathCreator();

  if (!deferredFrom) {
    return null;
  }

  const { parentRunID, parentFnSlug, parentRun } = deferredFrom;

  return (
    <CollapsibleSection title="Deferred from">
      <div className="flex flex-row flex-wrap items-center justify-start gap-x-10 gap-y-4">
        <ElementWrapper label="Function">
          <LinkElement href={pathCreator.function({ functionSlug: parentFnSlug })}>
            {parentFnSlug}
          </LinkElement>
        </ElementWrapper>

        <ElementWrapper label="Run">
          <LinkElement href={pathCreator.runPopout({ runID: parentRunID })}>
            {parentRunID}
          </LinkElement>
        </ElementWrapper>

        <ElementWrapper label="Status">
          {parentRun ? <StatusCell status={parentRun.status} /> : <TextElement>—</TextElement>}
        </ElementWrapper>
      </div>
    </CollapsibleSection>
  );
};
