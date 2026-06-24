package actions

type DownloadDumpParams struct {
}

func (p DownloadDumpParams) Validate() error {
	return nil
}

func DownloadDump(p DownloadDumpParams) (string, error) {
	return "Dump downloaded not implemented", nil
}
