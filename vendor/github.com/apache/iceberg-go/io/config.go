// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package io

// Constants for S3 configuration options
const (
	S3Region          = "s3.region"
	S3SessionToken    = "s3.session-token"
	S3SecretAccessKey = "s3.secret-access-key"
	S3AccessKeyID     = "s3.access-key-id"
	S3EndpointURL     = "s3.endpoint"
	S3ProxyURI        = "s3.proxy-uri"
	// S3ConnectTimeout accepts seconds as a number, such as "60" or "60.0",
	// or a Go duration string, such as "5s".
	S3ConnectTimeout         = "s3.connect-timeout"
	S3SignerURI              = "s3.signer.uri"
	S3RemoteSigningEnabled   = "s3.remote-signing-enabled"
	S3ForceVirtualAddressing = "s3.force-virtual-addressing"
	S3ClientRegion           = "client.region"
	S3CompatMode             = "s3.compat-mode"
)

// Constants for GCS configuration options
const (
	GCSEndpoint       = "gcs.endpoint"
	GCSServiceHost    = "gcs.service.host"
	GCSKeyPath        = "gcs.keypath"
	GCSJSONKey        = "gcs.jsonkey"
	GCSCredType       = "gcs.credtype"
	GCSUseJSONAPI     = "gcs.usejsonapi"
	GCSOAuthToken     = "gcs.oauth2.token"
	GCSOAuthExpiresAt = "gcs.oauth2.token-expires-at"
	GCSNoAuth         = "gcs.no-auth"
)

// Constants for Azure configuration options
const (
	ADLSSasTokenPrefix         = "adls.sas-token."
	ADLSConnectionStringPrefix = "adls.connection-string."
	ADLSSharedKeyAccountName   = "adls.auth.shared-key.account.name"
	ADLSSharedKeyAccountKey    = "adls.auth.shared-key.account.key"
	ADLSEndpoint               = "adls.endpoint"
	ADLSProtocol               = "adls.protocol"

	// ADLSManagedIdentityEnabled uses ManagedIdentityCredential directly when set
	// to "true", rather than the broader DefaultAzureCredential chain, which already
	// tries managed identity among its other sources. Pair it with ADLSClientID to
	// select a user-assigned managed identity; on its own it uses the system-assigned
	// managed identity.
	//
	// It is evaluated only after shared-key credentials, a SAS-token property for
	// the target hostname, and a connection-string property for the target account
	// name, all of which take precedence over it; if any of them is configured,
	// this flag has no effect.
	//
	// This property is specific to this implementation. Neither the Java nor the
	// PyIceberg implementation defines an equivalent property, so catalog properties
	// that set it will not carry over to them.
	ADLSManagedIdentityEnabled = "adls.auth.managed-identity.enabled"

	// ADLSClientID selects a user-assigned managed identity by client ID. It only
	// takes effect when ADLSManagedIdentityEnabled is "true"; on its own it is ignored.
	//
	// PyIceberg recognizes the same "adls.client-id" key but uses it for service
	// principal authentication, alongside "adls.client-secret" and "adls.tenant-id",
	// which it requires to be set together; see
	// https://py.iceberg.apache.org/configuration/#azure-data-lake. This
	// implementation defines neither of those, so the key is only ever read as a
	// user-assigned managed identity ID here. Java's AzureProperties defines no
	// "adls.client-id" at all.
	ADLSClientID = "adls.client-id"

	// Not in use yet
	// ADLSReadBlockSize          = "adls.read.block-size-bytes"
	// ADLSWriteBlockSize         = "adls.write.block-size-bytes"
)
