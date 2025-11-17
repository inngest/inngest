import { Info } from "@inngest/components/Info/Info";
import { Link } from "@inngest/components/Link/NewLink";

export const EventTypesInfo = () => (
  <Info
    text="List of all Inngest event types in the current environment."
    action={
      <Link href={"https://www.inngest.com/docs/events"} target="_blank">
        Learn how events work
      </Link>
    }
  />
);
