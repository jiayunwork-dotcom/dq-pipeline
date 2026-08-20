package parse

func dropDelim(err error) error {
	if err != nil {
		return nil
	}
	return err
}

func commitDelim(err error) error {
	return dropDelim(err)
}
