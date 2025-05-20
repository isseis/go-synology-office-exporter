package synology_drive_exporter

import (
	"fmt"

	synd "github.com/isseis/go-synology-office-exporter/synology_drive_api"
)

// SessionInterface abstracts Synology session operations for export and testability.
type SessionInterface interface {
	// List retrieves a paginated list of items from the specified root directory.
	List(rootDirID synd.FileID, offset, limit int64) (*synd.ListResponse, error)

	// Export exports the specified file, performing format conversion if needed.
	Export(fileID synd.FileID) (*synd.ExportResponse, error)

	// TeamFolder retrieves a paginated list of team folders.
	TeamFolder(offset, limit int64) (*synd.TeamFolderResponse, error)

	// SharedWithMe retrieves a paginated list of files and folders shared with the user.
	SharedWithMe(offset, limit int64) (*synd.SharedWithMeResponse, error)
}

// listAllPaginated is a generic function that handles pagination for list operations.
// It accepts a function that fetches a page of items and returns them along with the total count.
type listPageFunc[T any] func(offset, limit int64) (items []T, total int64, err error)

// listAllPaginated handles pagination for list operations
func listAllPaginated[T any](fetchPage listPageFunc[T]) ([]T, error) {
	var allItems []T
	var totalItems int64

	for offset := int64(0); ; offset += synd.DefaultMaxPageSize {
		items, total, err := fetchPage(offset, synd.DefaultMaxPageSize)
		if err != nil {
			return nil, fmt.Errorf("error listing items at offset %d: %w", offset, err)
		}

		allItems = append(allItems, items...)

		// Update total items on first page if available
		if offset == 0 && total > 0 {
			totalItems = total
		}

		// Stop if we've got all items or if no more items are returned
		if int64(len(allItems)) >= totalItems || len(items) == 0 {
			break
		}
	}

	return allItems, nil
}

// listAll retrieves all items from a directory by making multiple paginated requests.
func listAll(s SessionInterface, rootDirID synd.FileID) ([]*synd.ResponseItem, error) {
	return listAllPaginated(func(offset, limit int64) ([]*synd.ResponseItem, int64, error) {
		resp, err := s.List(rootDirID, offset, limit)
		if err != nil {
			return nil, 0, err
		}
		return resp.Items, resp.Total, nil
	})
}

// teamFoldersAll retrieves all team folders by making multiple paginated requests.
func teamFoldersAll(s SessionInterface) ([]*synd.TeamFolderResponseItem, error) {
	return listAllPaginated(func(offset, limit int64) ([]*synd.TeamFolderResponseItem, int64, error) {
		resp, err := s.TeamFolder(offset, limit)
		if err != nil {
			return nil, 0, fmt.Errorf("error listing team folders: %w", err)
		}
		return resp.Items, resp.Total, nil
	})
}

// sharedWithMeAll retrieves all shared items by making multiple paginated requests.
func sharedWithMeAll(s SessionInterface) ([]*synd.ResponseItem, error) {
	return listAllPaginated(func(offset, limit int64) ([]*synd.ResponseItem, int64, error) {
		resp, err := s.SharedWithMe(offset, limit)
		if err != nil {
			return nil, 0, fmt.Errorf("error listing shared items: %w", err)
		}
		return resp.Items, resp.Total, nil
	})
}
