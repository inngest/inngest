package queue

type PartitionInspectionResult struct {
	QueuePartition       *QueuePartition
	QueueShadowPartition *QueueShadowPartition

	Paused            bool `json:"paused"`
	Migrate           bool `json:"migrate"`
	AccountInProgress int  `json:"acct_in_progress"`
	Ready             int  `json:"ready"`
	InProgress        int  `json:"in_progress"`
	Future            int  `json:"future"`
	Backlogs          int  `json:"backlogs"`
}
