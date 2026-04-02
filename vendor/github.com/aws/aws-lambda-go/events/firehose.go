// Copyright 2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.

package events

// KinesisFirehoseEvent represents the input event from Amazon Kinesis Firehose. It is used as the input parameter.
type KinesisFirehoseEvent struct {
	InvocationID           string                       `json:"invocationId"`
	DeliveryStreamArn      string                       `json:"deliveryStreamArn"`      //nolint: staticcheck
	SourceKinesisStreamArn string                       `json:"sourceKinesisStreamArn"` //nolint: staticcheck
	Region                 string                       `json:"region"`
	Records                []KinesisFirehoseEventRecord `json:"records"`
}

type KinesisFirehoseEventRecord struct {
	RecordID                      string                        `json:"recordId"`
	ApproximateArrivalTimestamp   MilliSecondsEpochTime         `json:"approximateArrivalTimestamp"`
	Data                          []byte                        `json:"data"`
	KinesisFirehoseRecordMetadata KinesisFirehoseRecordMetadata `json:"kinesisRecordMetadata"`
}

// Constants used for describing the transformation result
const (
	KinesisFirehoseTransformedStateOk               = "Ok"
	KinesisFirehoseTransformedStateDropped          = "Dropped"
	KinesisFirehoseTransformedStateProcessingFailed = "ProcessingFailed"
)

// KinesisFirehoseOTFOperation represents the operation to apply on the record during on-the-fly record routing.
type KinesisFirehoseOTFOperation string

const (
	KinesisFirehoseOTFOperationInsert KinesisFirehoseOTFOperation = "insert"
	KinesisFirehoseOTFOperationUpdate KinesisFirehoseOTFOperation = "update"
	KinesisFirehoseOTFOperationDelete KinesisFirehoseOTFOperation = "delete"
)

type KinesisFirehoseResponse struct {
	Records []KinesisFirehoseResponseRecord `json:"records"`
}

type KinesisFirehoseResponseRecord struct {
	RecordID string                                `json:"recordId"`
	Result   string                                `json:"result"` // The status of the transformation. May be TransformedStateOk, TransformedStateDropped or TransformedStateProcessingFailed
	Data     []byte                                `json:"data"`
	Metadata KinesisFirehoseResponseRecordMetadata `json:"metadata"`
}

type KinesisFirehoseResponseRecordMetadata struct {
	PartitionKeys map[string]string                        `json:"partitionKeys"`
	OTFMetadata   KinesisFirehoseResponseRecordOTFMetadata `json:"otfMetadata"`
}

type KinesisFirehoseResponseRecordOTFMetadata struct {
	DestinationDatabaseName string                      `json:"destinationDatabaseName"`
	DestinationTableName    string                      `json:"destinationTableName"`
	Operation               KinesisFirehoseOTFOperation `json:"operation"` // The Operation field must have one of the following values – insert, update, or delete.
}

type KinesisFirehoseRecordMetadata struct {
	ShardID                     string                `json:"shardId"`
	PartitionKey                string                `json:"partitionKey"`
	SequenceNumber              string                `json:"sequenceNumber"`
	SubsequenceNumber           int64                 `json:"subsequenceNumber"`
	ApproximateArrivalTimestamp MilliSecondsEpochTime `json:"approximateArrivalTimestamp"`
}
