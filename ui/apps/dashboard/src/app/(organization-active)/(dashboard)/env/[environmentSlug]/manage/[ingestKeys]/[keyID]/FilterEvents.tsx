'use client';

import { useContext, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import ToggleGroup from '@inngest/components/ToggleGroup/ToggleGroup';
import { toast } from 'sonner';

import CodeEditor from '@/components/Textarea/CodeEditor';
import { Context } from './Context';
import { FilterEditor } from './FilterEditor';

type FilterEventsProps = {
  keyID: string;
  keyName: string | null;
  filter: {
    type: 'allow' | 'deny';
    ips: string[] | null;
    events: string[] | null;
  };
};

export default function FilterEvents({ keyID, filter }: FilterEventsProps) {
  const [newFilter, setNewFilter] = useState(filter);
  const [isDisabled, setDisabled] = useState(true);
  const { save } = useContext(Context);
  const router = useRouter();

  function validateSubmit(nextValue: {}) {
    if (JSON.stringify(nextValue) === JSON.stringify(filter)) {
      setDisabled(true);
    } else {
      setDisabled(false);
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (isDisabled) return;
    save({
      id: keyID,
      filter: newFilter,
    }).then((result) => {
      if (result.error) {
        toast.error('Event key filter has not been updated');
      } else {
        toast.success('Event key filter updated');
        router.refresh();
      }
    });
  }

  function handleClick(id: String) {
    const nextValue = { ...newFilter, type: id as 'allow' | 'deny' };
    validateSubmit(nextValue);

    if (newFilter.type !== id) {
      setNewFilter(nextValue);
    }
  }

  function handleCodeChange(name: 'events' | 'ips', code: string) {
    const trimmedValue = code.trim();
    const nextValueFull = { ...newFilter, [name]: trimmedValue.split('\n') };

    setNewFilter(nextValueFull);
    validateSubmit(nextValueFull);
  }

  return (
    <form className="pt-3" onSubmit={handleSubmit}>
      <h2 className="pb-1 text-lg font-semibold">Filter Events</h2>
      <p className="text-subtle text-sm">
        Filtering allows you to specify allow or deny lists for event names and/or IP addresses.
      </p>
      <p className="text-subtle text-sm">
        Allowlists only allow specified values, whereas denylists allow all but the specified
        values.
      </p>
      <p className="text-subtle mt-2 text-sm font-bold">
        You cannot use both allowlists and denylists simultaneously.
      </p>
      <div className="my-5 inline-block">
        <ToggleGroup
          type="single"
          defaultValue={newFilter.type}
          size="small"
          onValueChange={handleClick}
        >
          <ToggleGroup.Item value="allow">Allowlist</ToggleGroup.Item>
          <ToggleGroup.Item value="deny">Denylist</ToggleGroup.Item>
        </ToggleGroup>
      </div>
      <div className="mb-5 flex gap-5">
        <FilterEditor filter="events" list={newFilter.type}>
          <CodeEditor
            language="plaintext"
            initialCode={(filter.events || []).join('\n')}
            onCodeChange={(code) => handleCodeChange('events', code)}
          />
        </FilterEditor>
        <FilterEditor filter="IPs" list={newFilter.type}>
          <CodeEditor
            language="plaintext"
            initialCode={(newFilter.ips || []).join('\n')}
            onCodeChange={(code) => handleCodeChange('ips', code)}
          />
        </FilterEditor>
      </div>
      <div className="flex justify-end">
        <Button disabled={isDisabled} type="submit" label="Save filter changes" />
      </div>
    </form>
  );
}
