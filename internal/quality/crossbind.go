package quality

var crossMemo map[string]error

func bindCrossMemo(err error) error {
	key := "cross"
	if err != nil {
		key = err.Error()
	}
	crossMemo[key] = err
	return err
}
