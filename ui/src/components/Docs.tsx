export const Docs = () => {
  /**
   * When link seleted.
   * Make URL with https://www.inngest.com as base, and URL as path
   * Ensure url.pathname starts with /docs
   * If not, open in new tab
   */

  return <iframe src="https://inngest.com/docs" className="w-full h-full" />;
};
