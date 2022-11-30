import { useAppSelector } from "../store/hooks";

export const Docs = () => {
  /**
   * When link seleted.
   * Make URL with https://www.inngest.com as base, and URL as path
   * Ensure url.pathname starts with /docs
   * If not, open in new tab
   */
  const path = useAppSelector((state) => state.global.docsPath);

  return (
    <iframe
      src={`https://inngest.com/docs${path || ""}`}
      className="w-full h-full"
    />
  );
};
