package rbac

import "errors"

func Enforce(role string, allowed ...string) error {
	for _, a := range allowed {
		if role == a {
			return nil
		}
	}
	return errors.New("forbidden")
}
