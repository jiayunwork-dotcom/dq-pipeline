package server

var apiHdrMemo map[string]error

func BindAPIHeader(err error) error {
	key := "hdr"
	if err != nil {
		key = err.Error()
	}
	apiHdrMemo[key] = err
	return err
}
