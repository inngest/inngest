import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { useDefaultEnvironment } from '@/queries';
import type VercelIntegration from './VercelIntegration';

const CreateVercelAppDocument = graphql(`
  mutation CreateVercelApp($input: CreateVercelAppInput!) {
    createVercelApp(input: $input) {
      success
    }
  }
`);

const UpdateVercelAppDocument = graphql(`
  mutation UpdateVercelApp($input: UpdateVercelAppInput!) {
    updateVercelApp(input: $input) {
      success
    }
  }
`);

const RemoveVercelAppDocument = graphql(`
  mutation RemoveVercelApp($input: RemoveVercelAppInput!) {
    removeVercelApp(input: $input) {
      success
    }
  }
`);

export default function useUpdateVercelIntegration(initialVercelIntegration: VercelIntegration) {
  const [{ data: environment }] = useDefaultEnvironment();

  const [, createVercelApp] = useMutation(CreateVercelAppDocument);
  const [, updateVercelApp] = useMutation(UpdateVercelAppDocument);
  const [, removeVercelApp] = useMutation(RemoveVercelAppDocument);

  return (updatedVercelIntegration: VercelIntegration) => {
    const initialProjects = initialVercelIntegration.projects;
    const projects = updatedVercelIntegration.projects;

    const projectsToCreate = projects.filter((project) => {
      const initialProject = initialProjects.find(
        (initialProject) => initialProject.id === project.id
      );
      const projectIsNew = !initialProject;
      const projectHasBeenEnabled = !projectIsNew && !initialProject.isEnabled && project.isEnabled;
      return projectIsNew || projectHasBeenEnabled;
    });

    const projectsToUpdate = projects.filter((project) => {
      const initialProject = initialProjects.find(
        (initialProject) => initialProject.id === project.id
      );
      return initialProject && project.isEnabled && project.servePath !== initialProject.servePath;
    });

    const projectsToRemove = projects.filter((project) => {
      const initialProject = initialProjects.find(
        (initialProject) => initialProject.id === project.id
      );
      const projectIsNew = !initialProject;
      const projectHasBeenDisabled =
        !projectIsNew && initialProject.isEnabled && !project.isEnabled;
      return !projectIsNew && projectHasBeenDisabled;
    });

    const createVercelAppPromises = projectsToCreate.map((project) =>
      createVercelApp({
        input: {
          path: project.servePath,
          projectID: project.id,
          workspaceID: environment!.id,
        },
      })
    );

    const updateVercelAppPromises = projectsToUpdate.map((project) => {
      return updateVercelApp({
        input: {
          projectID: project.id,
          path: project.servePath ?? '',
        },
      });
    });

    const removeVercelAppPromises = projectsToRemove.map((project) =>
      removeVercelApp({
        input: {
          projectID: project.id,
          workspaceID: environment!.id,
        },
      })
    );

    return Promise.all([
      ...createVercelAppPromises,
      ...updateVercelAppPromises,
      ...removeVercelAppPromises,
    ]);
  };
}
