package quality

import "dq-pipeline/internal/schema"

func rangeApplies(sch *schema.Schema, col string) bool {
	if sch == nil {
		return false
	}
	c := sch.ColumnByName(col)
	if c == nil {
		return false
	}
	return c.Type == schema.TypeInt || c.Type == schema.TypeFloat
}
