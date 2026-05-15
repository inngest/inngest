package output

import (
	"fmt"

	"github.com/inngest/inngest/pkg/constraintapi"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func TextCheckConstraints(combined *pb.CheckConstraintsResponse) error {
	if combined == nil {
		fmt.Println("empty response")
		return nil
	}

	req, err := constraintapi.CapacityCheckRequestFromProto(combined.Request)
	if err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	resp := constraintapi.CapacityCheckResponseFromProto(combined.Response)

	w := NewTextWriter()

	// Write index summary first
	if err := w.WriteOrdered(OrderedData(
		"Account ID", req.AccountID,
		"Env ID", req.EnvID,
		"Function ID", req.FunctionID,
		"Available", resp.AvailableCapacity,
		"Fairness Reduction", resp.FairnessReduction,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	if err := w.Flush(); err != nil {
		return err
	}

	// // Write constraint usage information
	if len(resp.Usage) > 0 {
		fmt.Printf("\nUsage:\n")

		for i, usage := range resp.Usage {
			fmt.Printf("\n  Usage %d:\n", i+1)
			fmt.Printf("    Config:         %s\n", usage.Constraint.PrettyStringConfig(req.Configuration))
			fmt.Printf("    Constraint:     %s\n", usage.Constraint.PrettyString())
			fmt.Printf("    Usage:          %d\n", usage.Used)
			fmt.Printf("    Limit:          %d\n", usage.Limit)
		}
	} else {
		fmt.Printf("\nNo usage returned\n")
	}

	return nil
}
