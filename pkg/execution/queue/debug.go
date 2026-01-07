package queue

type PartitionInspectionResult struct {
	QueuePartition       *QueuePartition
	QueueShadowPartition *QueueShadowPartition

	Paused            bool `json:"paused"`
	Migrate           bool `json:"migrate"`
	AccountActive     int  `json:"acct_active"`
	AccountInProgress int  `json:"acct_in_progress"`
	Ready             int  `json:"ready"`
	InProgress        int  `json:"in_progress"`
	Active            int  `json:"active"`
	Future            int  `json:"future"`
	Backlogs          int  `json:"backlogs"`
}
