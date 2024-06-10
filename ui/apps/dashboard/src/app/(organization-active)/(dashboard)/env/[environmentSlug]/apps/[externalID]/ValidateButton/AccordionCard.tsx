import * as Accordion from '@radix-ui/react-accordion';
import { RiArrowDownSLine } from '@remixicon/react';

export function AccordionCard({
  children,
  type = 'multiple',
}: React.PropsWithChildren<{ type?: React.ComponentProps<typeof Accordion.Root>['type'] }>) {
  return (
    <Accordion.Root className="rounded-md border border-gray-300" type={type}>
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
    <Accordion.Item className="border-t border-gray-300 first:border-t-0" value={value}>
      <Accordion.Trigger
        asChild
        className="group w-full border-b border-gray-300 p-2 text-left text-sm text-gray-600 data-[state=closed]:border-b-0 data-[state=open]:bg-gray-100"
      >
        <button className="flex w-full">
          <RiArrowDownSLine
            aria-hidden
            className="mr-1 h-5 transition-transform duration-300 ease-[cubic-bezier(0.87,_0,_0.13,_1)] group-data-[state=closed]:-rotate-90"
          />

          <div className="grow text-left">{header}</div>
        </button>
      </Accordion.Trigger>

      <Accordion.Content className="max-h-64 overflow-scroll p-4">{children}</Accordion.Content>
    </Accordion.Item>
  );
}

AccordionCard.Item = Item;
