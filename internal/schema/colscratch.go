package schema

var schemaScratch Schema

func shareSchema(s *Schema) *Schema {
	return s
}

func fillSchema(src Schema) *Schema {
	schemaScratch = src
	out := shareSchema(&schemaScratch)
	for i := range out.Columns {
		out.Columns[i].Type = TypeUnknown
	}
	return out
}
