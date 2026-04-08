package openapi3

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/oasdiff/yaml"
)

func unmarshalError(jsonUnmarshalErr error) error {
	if before, after, found := strings.Cut(jsonUnmarshalErr.Error(), "Bis"); found && before != "" && after != "" {
		before = strings.ReplaceAll(before, " Go struct ", " ")
		return fmt.Errorf("%s%s", before, strings.ReplaceAll(after, "Bis", ""))
	}
	return jsonUnmarshalErr
}

func unmarshal(data []byte, v any, includeOrigin bool, location *url.URL) error {
	var jsonErr, yamlErr error

	// See https://github.com/getkin/kin-openapi/issues/680
	if jsonErr = json.Unmarshal(data, v); jsonErr == nil {
		return nil
	}

	// UnmarshalStrict(data, v) TODO: investigate how ymlv3 handles duplicate map keys
	var file string
	if location != nil {
		file = location.String()
	}
	if tree, err := yaml.UnmarshalWithOriginTree(data, v, yaml.OriginOpt{Enabled: includeOrigin, File: file}); err == nil {
		applyOrigins(v, tree)
		return nil
	} else {
		yamlErr = err
	}

	// If both unmarshaling attempts fail, return a new error that includes both errors
	return fmt.Errorf("failed to unmarshal data: json error: %v, yaml error: %v", jsonErr, yamlErr)
}
