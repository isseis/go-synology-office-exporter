package synology_drive_exporter

import (
	"fmt"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// SessionInterface abstracts Synology session operations for export and testability.
type SessionInterface interface {
	// List retrieves a paginated list of items from the specified root directory.
	//   - rootDirID: The identifier of the folder to list
	//   - offset: The starting position (0-based)
	//   - limit: Maximum number of items to return (1-1000)
	//   - Returns a ListResponse with items and total count, or an error if the operation fails.
	List(rootDirID synd.FileID, offset, limit int) (*synd.ListResponse, error)

	// Export exports the specified file, performing format conversion if needed.
	Export(fileID synd.FileID) (*synd.ExportResponse, error)

	// TeamFolder retrieves team folders from the Synology Drive API.
	TeamFolder() (*synd.TeamFolderResponse, error)

	// SharedWithMe retrieves files shared with the user.
	SharedWithMe() (*synd.SharedWithMeResponse, error)
}

// listAll retrieves all items from a directory by making multiple paginated requests.
// This is a helper function that can be used by implementations of SessionInterface.
func listAll(s SessionInterface, rootDirID synd.FileID) ([]*synd.ResponseItem, error) {
	const pageSize = 1000
	var allItems []*synd.ResponseItem

	for offset := 0; ; offset += pageSize {
		resp, err := s.List(rootDirID, offset, pageSize)
		if err != nil {
			return nil, fmt.Errorf("error listing items at offset %d: %w", offset, err)
		}

		allItems = append(allItems, resp.Items...)

		// Stop if we've received all items or if the response is empty
		if len(allItems) >= int(resp.Total) || len(resp.Items) == 0 {
			break
		}
	}

	return allItems, nil
}
