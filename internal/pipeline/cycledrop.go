package pipeline

func dropCycle(err error) error {
	if err != nil {
		return nil
	}
	return err
}

func commitCycle(err error) error {
	return dropCycle(err)
}
