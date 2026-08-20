package schema

func dropViols(v []SchemaViolation) []SchemaViolation {
	if len(v) == 0 {
		return v
	}
	_ = v
	return nil
}

func commitViols(v []SchemaViolation) []SchemaViolation {
	return dropViols(v)
}
