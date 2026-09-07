package auth

import "strings"

// UserStatus is a User's account status, independent of Role (which
// decides what an active User is allowed to do; UserStatus decides
// whether they may authenticate at all). It is a distinct type, not a
// raw string, following the exact pattern of Role and
// customer.CustomerStatus.
//
// This is deliberately a flat, two-value status, not Customer's
// three-value Active/Inactive/Archived: there is no "Archived" concept
// for a User yet, since nothing today needs to distinguish "deactivated,
// might come back" from "gone for good" — Users are never deleted (see
// UserRepository's doc comment), so Inactive already means "cannot log
// in, record kept."
type UserStatus string

// The two defined statuses. There is no zero-value/default status — an
// empty UserStatus is invalid — so UserStatus is effectively required on
// every User (see User.Validate in validate.go), for the same reason
// Role has no default value.
const (
	UserStatusActive   UserStatus = "Active"
	UserStatusInactive UserStatus = "Inactive"
)

// userStatusOrder is the authoritative, ordered set of valid statuses. It
// backs both Valid and validation error messages so the two can never
// disagree.
var userStatusOrder = []UserStatus{
	UserStatusActive,
	UserStatusInactive,
}

// Valid reports whether s is one of the two defined UserStatus values.
func (s UserStatus) Valid() bool {
	for _, v := range userStatusOrder {
		if s == v {
			return true
		}
	}
	return false
}

// String implements fmt.Stringer.
func (s UserStatus) String() string {
	return string(s)
}

// userStatusNames renders the defined statuses as a comma-separated
// list, for use in validation error messages.
func userStatusNames() string {
	names := make([]string, len(userStatusOrder))
	for i, s := range userStatusOrder {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}
