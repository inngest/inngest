import * as Accordion from '@radix-ui/react-accordion';
import { RiArrowDownSLine } from '@remixicon/react';

export function AccordionCard({
  children,
  type = 'multiple',
}: React.PropsWithChildren<{ type?: React.ComponentProps<typeof Accordion.Root>['type'] }>) {
  return (
    <Accordion.Root className="rounded-md border border-slate-300" type={type}>
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
    <Accordion.Item className="border-t border-slate-300 first:border-t-0" value={value}>
      <Accordion.Trigger
        asChild
        className="group w-full border-b border-slate-300 px-4 py-2 text-left font-semibold data-[state=closed]:border-b-0"
      >
        <button className="flex w-full">
          <div className="grow text-left">{header}</div>

          <RiArrowDownSLine
            aria-hidden
            className="transition-transform duration-300 ease-[cubic-bezier(0.87,_0,_0.13,_1)] group-data-[state=open]:rotate-180"
          />
        </button>
      </Accordion.Trigger>

      <Accordion.Content className="p-4">{children}</Accordion.Content>
    </Accordion.Item>
  );
}

AccordionCard.Item = Item;
