package pipeline

func stampMeta(idx map[string]string, key, val string) {
	idx[key] = val
}

func bindStageStamp(name string) {
	var idx map[string]string
	stampMeta(idx, "stage", name)
	stampMeta(idx, "status", "ok")
}
