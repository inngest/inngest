import { graphql } from '@/gql';

export const GetSandboxSecretsDocument = graphql(`
  query GetSandboxSecrets($environmentID: ID!) {
    workspace(id: $environmentID) {
      id
      envSecrets {
        id
        name
        createdAt
        updatedAt
      }
    }
  }
`);

export const CreateSandboxSecretDocument = graphql(`
  mutation CreateSandboxSecret($input: CreateEnvSecretInput!) {
    createEnvSecret(input: $input) {
      id
      name
      createdAt
      updatedAt
    }
  }
`);

export const ReplaceSandboxSecretDocument = graphql(`
  mutation ReplaceSandboxSecret($input: UpdateEnvSecretValueInput!) {
    updateEnvSecretValue(input: $input) {
      id
      name
      createdAt
      updatedAt
    }
  }
`);

export const ArchiveSandboxSecretDocument = graphql(`
  mutation ArchiveSandboxSecret($environmentID: UUID!, $id: UUID!) {
    archiveEnvSecret(workspaceID: $environmentID, id: $id)
  }
`);

export type SandboxSecret = {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
};

export const secretQueryContext = { additionalTypenames: ['EnvSecret'] };
