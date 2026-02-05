import { useQuery } from '@connectrpc/connect-query';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { fetchAccountEnvs } from '@inngest/components/proto/api/v2/service-V2_connectquery';
import {
  EnvType as ProtoEnvType,
  type Env as ProtoEnv,
} from '@inngest/components/proto/api/v2/service_pb';
import { EnvironmentType, type Environment } from '@/utils/environments';

//
// Map proto EnvType to dashboard EnvironmentType
const mapProtoEnvType = (type: ProtoEnvType): EnvironmentType => {
  switch (type) {
    case ProtoEnvType.PRODUCTION:
      return EnvironmentType.Production;
    case ProtoEnvType.TEST:
      return EnvironmentType.Test;
    case ProtoEnvType.BRANCH:
      return EnvironmentType.BranchChild;
    default:
      return EnvironmentType.Test;
  }
};

//
// Transform proto Env to dashboard Environment type
const protoEnvToEnvironment = (env: ProtoEnv): Environment => ({
  id: env.id,
  name: env.name,
  type: mapProtoEnvType(env.type),
  createdAt: env.createdAt ? timestampDate(env.createdAt).toISOString() : '',
  isArchived: env.isArchived,
  slug: env.slug,
  hasParent: Boolean(env.parentId),
  isAutoArchiveEnabled: env.isAutoArchiveEnabled,
  lastDeployedAt: env.lastDeployedAt
    ? timestampDate(env.lastDeployedAt).toISOString()
    : null,
  webhookSigningKey: '',
});

//
// React Query hook for fetching environments via ConnectRPC
export const useEnvironmentsGrpc = (options?: { enabled?: boolean }) => {
  const query = useQuery(
    fetchAccountEnvs,
    {},
    {
      staleTime: 30000,
      enabled: options?.enabled ?? true,
    },
  );

  return {
    ...query,
    data: query.data?.data.map(protoEnvToEnvironment) ?? [],
  };
};
