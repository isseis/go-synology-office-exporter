package synology_drive_api

import "errors"

type ExportResponse struct{}

func (s *SynologySession) Export(fileID FileID) (*ExportResponse, error) {
	return nil, errors.New("not implemented")
}
