//go:generate go run github.com/dmarkham/enumer -trimprefix=CancelReason -type=CancelReason -json -text -gqlgen

package enums

type CancelReason int

const (
	CancelReasonNone CancelReason = iota
	
	// User-initiated
	CancelReasonManualAPI         // Via REST API
	CancelReasonManualUI          // Via dashboard
	CancelReasonManualTest        // Via test environment
	CancelReasonBulkOperation     // Bulk cancel operation
	
	// System-initiated
	CancelReasonStartTimeout   // Function didn't start in time
	CancelReasonFinishTimeout  // Function didn't complete in time
	CancelReasonHTTPTimeout    // HTTP request to function timed out
	CancelReasonSingleton      // Replaced by newer singleton run
	
	// Event-driven
	CancelReasonEventMatch     // Cancel event matched expression
	CancelReasonEventTimeout   // Wait-for-event step timed out
	CancelReasonInvokeTimeout  // Invoke function step timed out
	
	// System maintenance
	CancelReasonSystemDrain    // Function cancelled during system shutdown
	CancelReasonResourceLimit  // Memory/CPU limits exceeded
	CancelReasonInternalError  // System failure requiring cancellation
)