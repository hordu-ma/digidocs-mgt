package memory

import "context"

type PermissionRepository struct{}

func NewPermissionRepository() PermissionRepository {
	return PermissionRepository{}
}

func (r PermissionRepository) CanCreateDocument(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanUpdateDocument(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanDeleteDocument(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanUploadVersion(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanFlowDocument(context.Context, string, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanCreateHandover(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanUpdateHandoverItems(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanApplyHandover(context.Context, string, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanUploadDataAsset(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanManageDataAsset(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanCreateCodeRepository(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanManageCodeRepository(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r PermissionRepository) CanPushCodeRepository(context.Context, string, string, string) (bool, error) {
	return true, nil
}
