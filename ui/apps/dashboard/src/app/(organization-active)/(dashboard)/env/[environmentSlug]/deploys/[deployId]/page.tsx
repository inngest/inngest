import DeployCard from '@/components/Cards/DeployCard';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';

const GetDeployDocument = graphql(`
  query GetDeploy($deployID: ID!) {
    deploy(id: $deployID) {
      id
      appName
      authorID
      checksum
      createdAt
      error
      framework
      metadata
      sdkLanguage
      sdkVersion
      status
      url

      deployedFunctions {
        slug
        name
      }

      removedFunctions {
        slug
        name
      }
    }
  }
`);

type DeployDetailProps = {
  params: {
    deployId: string;
  };
};

export const runtime = 'nodejs';

export default async function DeployDetail({ params }: DeployDetailProps) {
  const { deploy } = await graphqlAPI.request(GetDeployDocument, {
    deployID: params.deployId,
  });

  return (
    <div className="flex h-full grow items-stretch overflow-y-scroll">
      <DeployCard {...deploy} />
    </div>
  );
}
