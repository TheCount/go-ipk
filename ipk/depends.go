package ipk

import (
	"fmt"
	"strings"
)

// Dependency relationships
const (
	RelStrictlyEarlier = "<<"
	RelEarlierEqual    = "<="
	RelExact           = "="
	RelLaterEqual      = ">="
	RelStrictlyLater   = ">>"
)

// VersionedDependency returns a versioned dependency.
// The name is the package name, version is the version string, and
// relation is the dependency relationship, "<<", "<=", "=", ">=", or ">>".
// If version is the empty string, relation is ignored and name is returned.
// Otherwise, if relation is the empty string, "=" is assumed.
func VersionedDependency(name, relation, version string) (string, error) {
	if !allowedPackageNames.MatchString(name) {
		return "", fmt.Errorf("Invalid package name: %s", name)
	}
	if version == "" {
		return name, nil
	}
	if relation == "" {
		relation = RelExact
	} else {
		if err := checkRelation(relation); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s (%s %s)", name, relation, version), nil
}

// DisjunctiveDependency creates a disjunctive dependency, i. e., a dependency
// where one of the given constituent dependencies must be fulfilled.
// The arguments must be versioned dependencies returned by VersionedDependency.
// The arguments are not checked for validity.
// If no arguments are given, the empty string is returned.
func DisjunctiveDependency(deps ...string) string {
	if len(deps) == 0 {
		return ""
	}
	result := strings.Builder{}
	result.WriteString(deps[0])
	for i := 1; i < len(deps); i++ {
		result.WriteString(" | ")
		result.WriteString(deps[i])
	}
	return result.String()
}

// ConjunctiveDependency creates a conjunctive dependency, i. e., a dependency
// where all of the given constituent dependencies must be fulfilled.
// The arguments must be dependencies returned by VersionedDependency,
// DisjunctiveDepenency, or ConjunctiveDependency. Empty strings are ignored.
// If no arguments are given or all arguments are the empty string,
// the empty string is returned.
func ConjunctiveDependency(deps ...string) string {
	for len(deps) > 0 && len(deps[0]) == 0 {
		deps = deps[1:]
	}
	if len(deps) == 0 {
		return ""
	}
	result := strings.Builder{}
	result.WriteString(deps[0])
	for i := 1; i < len(deps); i++ {
		if deps[i] == "" {
			continue
		}
		result.WriteString(", ")
		result.WriteString(deps[i])
	}
	return result.String()
}

// checkRelation checks whether rel is a valid relation.
func checkRelation(rel string) error {
	switch rel {
	case RelStrictlyEarlier, RelEarlierEqual, RelExact, RelLaterEqual,
		RelStrictlyLater:
		return nil
	default:
		return fmt.Errorf("Invalid relation: %s", rel)
	}
}
