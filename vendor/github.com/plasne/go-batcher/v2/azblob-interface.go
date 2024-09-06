package batcher

import (
	"context"
	"io"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

// This interface describes an Azure Storage Container that can be mocked.
type azureContainer interface {
	Create(context.Context, azblob.Metadata, azblob.PublicAccessType) (*azblob.ContainerCreateResponse, error)
	NewBlockBlobURL(string) azblob.BlockBlobURL
}

// This interface describes an Azure Storage Blob that can be mocked.
type azureBlob interface {
	Upload(context.Context, io.ReadSeeker, azblob.BlobHTTPHeaders, azblob.Metadata, azblob.BlobAccessConditions, azblob.AccessTierType, azblob.BlobTagsMap, azblob.ClientProvidedKeyOptions) (*azblob.BlockBlobUploadResponse, error)
	AcquireLease(context.Context, string, int32, azblob.ModifiedAccessConditions) (*azblob.BlobAcquireLeaseResponse, error)
}
