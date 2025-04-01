export const UserlandSpan = ({ userlandAttrs }: { userlandAttrs: string }) => {
  let attrs = null;

  try {
    attrs = JSON.parse(userlandAttrs);
  } catch (error) {
    console.info('Error parsing userlandAttrs', error);
  }

  return attrs ? (
    <div className="flex flex-col text-sm font-medium leading-tight">
      {/* TODO: just an example, map over a subset */}
      {attrs['url.full'] && (
        <div className="text-muted mt-2 flex flex-row items-center justify-start gap-2 text-xs">
          <div className="text-muted text-xs">URL:</div>
          <div className="truncate">{attrs['url.full']}</div>
        </div>
      )}
    </div>
  ) : null;
};
