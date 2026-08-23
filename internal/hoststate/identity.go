package hoststate

// RepositoryIDForRoot returns the current host-local opaque repository ID for
// one filesystem repository root. Cross-host logical identity is a separate
// federation concern and must not infer semantics from this value.
func RepositoryIDForRoot(root string) string {
	return repositoryID(root)
}

// RepositoryDisplayNameForRoot returns the current human-readable repository
// label derived from its filesystem location.
func RepositoryDisplayNameForRoot(root string) string {
	return repositoryDisplayName(root)
}
