'use client';

import { useContext, useState } from 'react';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';

import Button from '@/components/Button';
import GroupButton from '@/components/GroupButton/GroupButton';
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

export default function FilterEvents({ keyID, filter, keyName }: FilterEventsProps) {
  const [newFilter, setNewFilter] = useState(
    filter || [{ type: 'allow', events: null, ips: null }]
  );
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
    const nextValueEmpty = { ...newFilter, [name]: null };
    const nextValueFull = { ...newFilter, [name]: trimmedValue.split('\n') };

    if (trimmedValue === '') {
      setNewFilter(nextValueEmpty);
      validateSubmit(nextValueEmpty);
      return;
    }

    setNewFilter(nextValueFull);
    validateSubmit(nextValueFull);
  }

  return (
    <form className="pt-3" onSubmit={handleSubmit}>
      <h2 className="pb-1 text-lg font-semibold">Filter Events</h2>
      <p className="text-sm text-slate-500">
        Filtering allows you to specify allow or deny lists for event names and/or IP addresses.
      </p>
      <p className="text-sm text-slate-500">
        Allowlists only allow specified values, whereas denylists allow all but the specified
        values.
      </p>
      <div className="my-5 inline-block">
        <GroupButton
          title="Select allowlist or denyList to configure filters"
          options={[
            { name: 'Allowlist', id: 'allow' },
            { name: 'Denylist', id: 'deny' },
          ]}
          handleClick={handleClick}
          selectedOption={newFilter.type}
        />
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
        <Button disabled={isDisabled} type="submit">
          Save Filter Changes
        </Button>
      </div>
    </form>
  );
}
