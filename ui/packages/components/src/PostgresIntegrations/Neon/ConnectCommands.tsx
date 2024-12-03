import { useState } from 'react';
import CommandBlock, { type TabsProps } from '@inngest/components/CodeBlock/CommandBlock';

const PostgresCommandBlock = ({ tabs }: { tabs: TabsProps[] }) => {
  const [activeTab, setActiveTab] = useState(tabs[0]?.title || '');
  const currentTabContent = tabs.find((tab) => tab.title === activeTab) || tabs[0];

  return (
    <CommandBlock.Wrapper>
      <CommandBlock.Header>
        <CommandBlock.Tabs tabs={tabs} activeTab={activeTab} setActiveTab={setActiveTab} />
      </CommandBlock.Header>
      <CommandBlock currentTabContent={currentTabContent} />
    </CommandBlock.Wrapper>
  );
};

export const RoleCommand = () => (
  <PostgresCommandBlock
    tabs={[
      {
        title: 'Create role',
        content: 'CREATE USER inngest WITH REPLICATION',
        language: 'sql',
      },
    ]}
  />
);

export const AccessCommand = () => (
  <PostgresCommandBlock
    tabs={[
      {
        title: 'Access command',
        content: `GRANT USAGE ON SCHEMA public TO inngest;\n\GRANT SELECT ON ALL TABLES IN SCHEMA public TO inngest;\n\ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO inngest;`,
        language: 'sql',
      },
    ]}
  />
);

export const ReplicationSlotCommand = () => (
  <PostgresCommandBlock
    tabs={[
      {
        title: 'Replication slot command',
        content: "SELECT pg_create_logical_replication_slot('inngest_cdc', 'pgoutput');",
        language: 'sql',
      },
    ]}
  />
);

export const AlterTableReplicationCommandOne = () => (
  <PostgresCommandBlock
    tabs={[
      {
        title: 'Alter table - Default',
        content: 'ALTER TABLE <table_name> REPLICA IDENTITY FULL;',
        language: 'sql',
      },
    ]}
  />
);

export const AlterTableReplicationCommandTwo = () => (
  <PostgresCommandBlock
    tabs={[
      {
        title: 'Create publication',
        content: 'CREATE PUBLICATION inngest FOR ALL TABLES;',
        language: 'sql',
      },
    ]}
  />
);
