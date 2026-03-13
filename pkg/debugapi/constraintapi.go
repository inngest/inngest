package debugapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	cpb "github.com/inngest/inngest/proto/gen/constraintapi/v1"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func (d *debugAPI) CheckConstraints(ctx context.Context, req *cpb.CapacityCheckRequest) (*pb.CheckConstraintsResponse, error) {
	if req.AccountId == "" {
		req.AccountId = consts.DevServerAccountID.String()
	}

	if req.EnvId == "" {
		req.EnvId = consts.DevServerEnvID.String()
	}

	if req.Configuration == nil {
		config := constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: constraintapi.ConcurrencyConfig{
				AccountConcurrency: consts.DefaultConcurrencyLimit,
			},
		}

		constraints := []constraintapi.ConstraintItem{
			{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Scope: enums.ConcurrencyScopeAccount,
				},
			},
		}

		if req.FunctionId != "" {
			functionID, err := uuid.Parse(req.FunctionId)
			if err != nil {
				return nil, fmt.Errorf("invalid function ID: %w", err)
			}

			fn, err := d.db.GetFunctionByInternalUUID(ctx, functionID)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve function: %w", err)
			}

			var inngestFunction inngest.Function
			err = json.Unmarshal(fn.Config, &inngestFunction)
			if err != nil {
				return nil, fmt.Errorf("could not parse function config: %w", err)
			}

			config.FunctionVersion = inngestFunction.FunctionVersion

			if inngestFunction.Concurrency != nil {
				for _, c := range inngestFunction.Concurrency.Limits {
					if c.IsPartitionLimit() {
						config.Concurrency.FunctionConcurrency = c.Limit
						constraints = append(constraints, constraintapi.ConstraintItem{
							Kind: constraintapi.ConstraintKindConcurrency,
							Concurrency: &constraintapi.ConcurrencyConstraint{
								Scope: enums.ConcurrencyScopeFn,
							},
						})
						continue
					}

					config.Concurrency.CustomConcurrencyKeys = append(config.Concurrency.CustomConcurrencyKeys, constraintapi.CustomConcurrencyLimit{
						Scope:             c.Scope,
						Limit:             c.Limit,
						KeyExpressionHash: c.Hash,
						Mode:              enums.ConcurrencyModeStep,
					})
					// TODO: Allow to provide key in request
					// constraints = append(constraints, constraintapi.ConstraintItem{
					// 	Kind: constraintapi.ConstraintKindConcurrency,
					// 	Concurrency: &constraintapi.ConcurrencyConstraint{
					// 		Scope:             c.Scope,
					// 	Mode: enums.ConcurrencyModeStep,
					// 	KeyExpressionHash: c.Hash,
					// 	EvaluatedKeyHash: c.EvaluatedKey,
					//
					// 		// InProgressItemKey: kg.Concurrency("custom", fn.ID.String()),
					// 	},
					// })
				}
			}

			// TODO Parse rate limit and throttle
			// TODO: Run rate limit checks in separate request
		}

		req.Configuration = constraintapi.ConstraintConfigToProto(config)

		serializedConstraints := make([]*cpb.ConstraintItem, len(constraints))
		for i, c := range constraints {
			serializedConstraints[i] = constraintapi.ConstraintItemToProto(c)
		}
		req.Constraints = serializedConstraints
	}

	parsed, err := constraintapi.CapacityCheckRequestFromProto(req)
	if err != nil {
		return nil, fmt.Errorf("could not parse request: %w", err)
	}

	resp, userErr, err := d.cm.Check(ctx, parsed)
	if err != nil {
		return nil, fmt.Errorf("could not check constraints: %w", err)
	}

	if userErr != nil {
		return nil, fmt.Errorf("user err: %w", userErr)
	}

	serializedResp := constraintapi.CapacityCheckResponseToProto(resp)

	return &pb.CheckConstraintsResponse{
		Request:  req,
		Response: serializedResp,
	}, nil
}

func (d *debugAPI) GetAccountConcurrency(ctx context.Context, req *pb.GetAccountConcurrencyRequest) (*pb.GetAccountConcurrencyResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	count, err := d.cdb.GetAccountConcurrency(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("could not get account concurrency: %w", err)
	}

	return &pb.GetAccountConcurrencyResponse{InProgress: int32(count)}, nil
}

func (d *debugAPI) GetFunctionConcurrency(ctx context.Context, req *pb.GetFunctionConcurrencyRequest) (*pb.GetFunctionConcurrencyResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	functionID, err := uuid.Parse(req.FunctionId)
	if err != nil {
		return nil, fmt.Errorf("invalid function ID: %w", err)
	}

	count, err := d.cdb.GetFunctionConcurrency(ctx, accountID, functionID)
	if err != nil {
		return nil, fmt.Errorf("could not get function concurrency: %w", err)
	}

	return &pb.GetFunctionConcurrencyResponse{InProgress: int32(count)}, nil
}

func (d *debugAPI) CountAccountLeases(ctx context.Context, req *pb.CountAccountLeasesRequest) (*pb.CountAccountLeasesResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	count, err := d.cdb.CountAccountLeases(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("could not count account leases: %w", err)
	}

	return &pb.CountAccountLeasesResponse{Count: int32(count)}, nil
}

func (d *debugAPI) CountAccounts(ctx context.Context, req *pb.CountAccountsRequest) (*pb.CountAccountsResponse, error) {
	count, err := d.cdb.CountAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not count accounts: %w", err)
	}

	return &pb.CountAccountsResponse{Count: int32(count)}, nil
}
