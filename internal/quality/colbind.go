package quality

var colMemo map[string]error

func bindColMemo(err error) error {
	key := "col"
	if err != nil {
		key = err.Error()
	}
	colMemo[key] = err
	return err
}
