# Core API

The Core API service enables the Inngest system to be managed remotely.
Mainly, the Core API is used to manage Functions and Actions via the Inngest CLI.

## GraphQL Development

The API is a GraphQL interface. Making changes to the API interfaces should be done with the following steps:

1. Edit `resolvers.graphql` or `mutations.graphql` in this directory.
2. Create your new functions for your new resolvers and mutations using the `models` package and use struct names that match your GraphQL inputs and types.
3. Generate new models and validate your new stubbed out functions running `make gql` from the root directory of this project.
4. Modify your new functions and view newly generated structs in `graph/models/models_gen.go`

### Example

If you were to add a new GraphQL mutation:

```graphql
mutation {
  updateActionType(input: UpdateActionType): ActionType
}
input UpdateActionType { ... }
type ActionType { ... }
```

Your new function should be added to a related resolver file, e.g. `graph/resolvers/action_types.go` with with a stubbed out resolver:

```go
// Replace w/ *queryResolver if it's a not a mutation
func (r *mutationResolver) updateActionType(
  ctx context.Context,
  input models.UpdateActionType
) (*models.ActionType, error) {
  return nil, nil
}
```
