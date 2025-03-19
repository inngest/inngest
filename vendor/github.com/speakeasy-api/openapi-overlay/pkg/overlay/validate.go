package overlay

import (
	"fmt"
	"net/url"
	"strings"
)

type ValidationErrors []error

func (v ValidationErrors) Error() string {
	msgs := make([]string, len(v))
	for i, err := range v {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "\n")
}

func (v ValidationErrors) Return() error {
	if len(v) > 0 {
		return v
	}
	return nil
}

func (o *Overlay) Validate() error {
	errs := make(ValidationErrors, 0)
	if o.Version != "1.0.0" {
		errs = append(errs, fmt.Errorf("overlay version must be 1.0.0"))
	}

	if o.Info.Title == "" {
		errs = append(errs, fmt.Errorf("overlay info title must be defined"))
	}
	if o.Info.Version == "" {
		errs = append(errs, fmt.Errorf("overlay info version must be defined"))
	}

	if o.Extends != "" {
		_, err := url.Parse(o.Extends)
		if err != nil {
			errs = append(errs, fmt.Errorf("overlay extends must be a valid URL"))
		}
	}

	if len(o.Actions) == 0 {
		errs = append(errs, fmt.Errorf("overlay must define at least one action"))
	} else {
		for i, action := range o.Actions {
			if action.Target == "" {
				errs = append(errs, fmt.Errorf("overlay action at index %d target must be defined", i))
			}

			if action.Remove && !action.Update.IsZero() {
				errs = append(errs, fmt.Errorf("overlay action at index %d should not both set remove and define update", i))
			}
		}
	}

	return errs.Return()
}
