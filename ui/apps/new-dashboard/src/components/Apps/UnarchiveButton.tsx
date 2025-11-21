import { Button } from "@inngest/components/Button/NewButton";

type Props = {
  showArchive: () => void;
};

export function UnarchiveButton({ showArchive }: Props) {
  return (
    <>
      <Button onClick={showArchive} kind="danger" label="Unarchive app" />
    </>
  );
}
