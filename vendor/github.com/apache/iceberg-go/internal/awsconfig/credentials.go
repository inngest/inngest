// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package awsconfig

import (
	"errors"
	"fmt"
)

var ErrIncompleteStaticCredentials = errors.New("incomplete static AWS credentials")

// ValidateStaticCredentials ensures a session token is only configured with
// the complete key pair required by AWS static credential providers.
func ValidateStaticCredentials(keyName, secretName, tokenName, key, secret, token string) error {
	if key == "" && secret == "" && token != "" {
		return fmt.Errorf("%w: %s requires %s and %s", ErrIncompleteStaticCredentials, tokenName, keyName, secretName)
	}
	if (key == "") != (secret == "") {
		return fmt.Errorf("%w: %s and %s must be configured together", ErrIncompleteStaticCredentials, keyName, secretName)
	}

	return nil
}
