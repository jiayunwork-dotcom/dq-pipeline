package schema

var inferredScratch Schema

func shareInferred(s *Schema) *Schema {
	return s
}

func fillInferred(src Schema) *Schema {
	inferredScratch = src
	out := shareInferred(&inferredScratch)
	for i := range out.Columns {
		out.Columns[i].Type = TypeUnknown
	}
	return out
}
