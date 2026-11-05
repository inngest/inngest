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

package table

import (
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"strings"
)

// EncryptionKey represents an encryption key stored in table metadata (V3+).
type EncryptionKey struct {
	KeyID                string            `json:"key-id"`
	EncryptedKeyMetadata string            `json:"encrypted-key-metadata"`
	EncryptedByID        *string           `json:"encrypted-by-id,omitempty"`
	Properties           map[string]string `json:"properties,omitempty"`
}

func (e EncryptionKey) validate() error {
	trimmedKeyID := strings.TrimSpace(e.KeyID)
	if trimmedKeyID == "" {
		return errors.New("encryption key-id must be non-empty")
	}
	if trimmedKeyID != e.KeyID {
		return errors.New("encryption key-id must not have leading or trailing whitespace")
	}

	decodedMetadata, err := base64.StdEncoding.DecodeString(e.EncryptedKeyMetadata)
	if err != nil {
		return fmt.Errorf("encrypted key metadata must be valid base64: %w", err)
	}
	if len(decodedMetadata) == 0 {
		return errors.New("encrypted key metadata must be non-empty")
	}

	return nil
}

func (e EncryptionKey) Equals(other EncryptionKey) bool {
	if e.KeyID != other.KeyID {
		return false
	}

	if e.EncryptedKeyMetadata != other.EncryptedKeyMetadata {
		return false
	}

	if e.EncryptedByID != other.EncryptedByID { // skip the whole thing if both are nil
		if e.EncryptedByID == nil || other.EncryptedByID == nil {
			// if either is nil, the other is not due to previous if
			return false
		} else if *e.EncryptedByID != *other.EncryptedByID {
			// if neither is nil, we can dereference them both safely
			return false
		}
	}

	return maps.Equal(e.Properties, other.Properties)
}
