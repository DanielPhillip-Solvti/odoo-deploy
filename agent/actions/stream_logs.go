package actions

type StreamLogsParams struct {
}

func (p StreamLogsParams) Validate() error {
	return nil
}

func StreamLogs(p StreamLogsParams) (string, error) {
	return "Stream logs not implemented", nil
}
