import * as Accordion from '@radix-ui/react-accordion';
import { RiArrowDownSLine } from '@remixicon/react';

export function AccordionCard({
  children,
  type = 'multiple',
}: React.PropsWithChildren<{ type?: React.ComponentProps<typeof Accordion.Root>['type'] }>) {
  return (
    <Accordion.Root className="border-subtle rounded-md border" type={type}>
      {children}
    </Accordion.Root>
  );
}

function Item({
  children,
  header,
  value,
}: React.PropsWithChildren<{ header: React.ReactNode; value: string }>) {
  return (
    <Accordion.Item className="border-subtle border-t first:border-t-0" value={value}>
      <Accordion.Trigger
        asChild
        className="border-subtle text-basis group w-full border-b p-3 text-left text-sm data-[state=closed]:border-b-0 data-[state=open]:bg-gray-100"
      >
        <button className="flex w-full items-center gap-1">
          <RiArrowDownSLine
            aria-hidden
            className="text-disabled h-4 w-4 transition-transform duration-300 ease-[cubic-bezier(0.87,_0,_0.13,_1)] group-data-[state=closed]:-rotate-90"
          />

          <div className="grow text-left">{header}</div>
        </button>
      </Accordion.Trigger>

      <Accordion.Content className="max-h-64 overflow-scroll p-4">{children}</Accordion.Content>
    </Accordion.Item>
  );
}

AccordionCard.Item = Item;
