package quality

var crossMemo map[string]error

func bindCrossMemo(err error) error {
	if crossMemo == nil {
		crossMemo = make(map[string]error)
	}
	key := "cross"
	if err != nil {
		key = err.Error()
	}
	crossMemo[key] = err
	return err
}
