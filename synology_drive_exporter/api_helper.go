package synology_drive_exporter

import (
	"fmt"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// SessionInterface abstracts Synology session operations for export and testability.
type SessionInterface interface {
	// List retrieves a paginated list of items from the specified root directory.
	List(rootDirID synd.FileID, offset, limit int) (*synd.ListResponse, error)

	// Export exports the specified file, performing format conversion if needed.
	Export(fileID synd.FileID) (*synd.ExportResponse, error)

	// TeamFolder retrieves a paginated list of team folders.
	TeamFolder(offset, limit int) (*synd.TeamFolderResponse, error)

	// SharedWithMe retrieves a paginated list of files and folders shared with the user.
	SharedWithMe(offset, limit int) (*synd.SharedWithMeResponse, error)
}

// listAll retrieves all items from a directory by making multiple paginated requests.
func listAll(s SessionInterface, rootDirID synd.FileID) ([]*synd.ResponseItem, error) {
	var allItems []*synd.ResponseItem
	var totalItems int
	for offset := 0; ; offset += synd.DefaultPageSize {
		resp, err := s.List(rootDirID, offset, synd.DefaultPageSize)
		if err != nil {
			return nil, fmt.Errorf("error listing items at offset %d: %w", offset, err)
		}
		allItems = append(allItems, resp.Items...)
		if offset == 0 && resp.Total > 0 {
			totalItems = int(resp.Total)
		}
		if len(allItems) >= totalItems || len(resp.Items) == 0 {
			break
		}
	}
	return allItems, nil
}

// listAllTeamFolders retrieves all team folders by making multiple paginated requests.
func listAllTeamFolders(s SessionInterface) ([]*synd.TeamFolderResponseItem, error) {
	var allItems []*synd.TeamFolderResponseItem
	var totalItems int
	for offset := 0; ; offset += synd.DefaultPageSize {
		resp, err := s.TeamFolder(offset, synd.DefaultPageSize)
		if err != nil {
			return nil, fmt.Errorf("error listing team folders at offset %d: %w", offset, err)
		}
		allItems = append(allItems, resp.Items...)
		if offset == 0 && resp.Total > 0 {
			totalItems = int(resp.Total)
		}
		if len(allItems) >= totalItems || len(resp.Items) == 0 {
			break
		}
	}
	return allItems, nil
}

// listAllSharedWithMe retrieves all shared items by making multiple paginated requests.
func listAllSharedWithMe(s SessionInterface) ([]*synd.ResponseItem, error) {
	var allItems []*synd.ResponseItem
	var totalItems int
	for offset := 0; ; offset += synd.DefaultPageSize {
		resp, err := s.SharedWithMe(offset, synd.DefaultPageSize)
		if err != nil {
			return nil, fmt.Errorf("error listing shared items at offset %d: %w", offset, err)
		}
		allItems = append(allItems, resp.Items...)
		if offset == 0 && resp.Total > 0 {
			totalItems = int(resp.Total)
		}
		if len(allItems) >= totalItems || len(resp.Items) == 0 {
			break
		}
	}
	return allItems, nil
}
