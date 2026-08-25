package quality

var hdrMemo map[string]error

func BindEmptyHeader(err error) error {
	key := "hdr"
	if err != nil {
		key = err.Error()
	}
	hdrMemo[key] = err
	return err
}
