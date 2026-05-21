package activitylog

import (
	"context"
	"fmt"
)

// =============================================================================
// Auth
// =============================================================================

// AuthAdapter satisfies the auth package's ActivityLogger interface.
type AuthAdapter struct {
	svc *Service
}

func NewAuthAdapter(svc *Service) *AuthAdapter { return &AuthAdapter{svc: svc} }

func (a *AuthAdapter) LogLogin(ctx context.Context, userID, userName, email string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:   "User Login",
		Detail:   fmt.Sprintf("%s (%s) logged in", userName, email),
		Type:     TypeInfo,
		UserID:   userID,
		Category: "auth",
	})
}

// =============================================================================
// Material Requests
// =============================================================================

// MaterialRequestsAdapter satisfies the materialrequests package's ActivityLogger interface.
type MaterialRequestsAdapter struct {
	svc *Service
}

func NewMaterialRequestsAdapter(svc *Service) *MaterialRequestsAdapter {
	return &MaterialRequestsAdapter{svc: svc}
}

func (a *MaterialRequestsAdapter) LogRequestCreated(ctx context.Context, requesterID, requesterName, requestNo, requestType, priority string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Material Request Submitted",
		Detail:       fmt.Sprintf("%s submitted %s (%s) with %s priority", requesterName, requestNo, requestType, priority),
		Type:         TypeInfo,
		UserID:       requesterID,
		ResourceType: "material_request",
		ResourceID:   requestNo,
		Category:     "material-request",
	})
}

func (a *MaterialRequestsAdapter) LogRequestApproved(ctx context.Context, approverID, approverName, requesterID, requestNo string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Material Request Approved",
		Detail:       fmt.Sprintf("%s approved %s", approverName, requestNo),
		Type:         TypeSuccess,
		UserID:       approverID,
		ResourceType: "material_request",
		ResourceID:   requestNo,
		Category:     "material-request",
	})
}

func (a *MaterialRequestsAdapter) LogRequestRejected(ctx context.Context, approverID, approverName, requesterID, requestNo string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Material Request Rejected",
		Detail:       fmt.Sprintf("%s rejected %s", approverName, requestNo),
		Type:         TypeWarning,
		UserID:       approverID,
		ResourceType: "material_request",
		ResourceID:   requestNo,
		Category:     "material-request",
	})
}

// =============================================================================
// Materials
// =============================================================================

// MaterialsAdapter satisfies the materials package's ActivityLogger interface.
type MaterialsAdapter struct {
	svc *Service
}

func NewMaterialsAdapter(svc *Service) *MaterialsAdapter {
	return &MaterialsAdapter{svc: svc}
}

func (a *MaterialsAdapter) LogMaterialCreated(ctx context.Context, actorID, materialID, sku, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Material Created",
		Detail:       fmt.Sprintf("Material %s (%s) ditambahkan", sku, name),
		Type:         TypeSuccess,
		UserID:       actorID,
		ResourceType: "material",
		ResourceID:   materialID,
		Category:     "materials",
	})
}

func (a *MaterialsAdapter) LogMaterialUpdated(ctx context.Context, actorID, materialID, sku, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Material Updated",
		Detail:       fmt.Sprintf("Material %s (%s) diperbarui", sku, name),
		Type:         TypeInfo,
		UserID:       actorID,
		ResourceType: "material",
		ResourceID:   materialID,
		Category:     "materials",
	})
}

func (a *MaterialsAdapter) LogMaterialDeleted(ctx context.Context, actorID, materialID, sku, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Material Deleted",
		Detail:       fmt.Sprintf("Material %s (%s) dihapus", sku, name),
		Type:         TypeWarning,
		UserID:       actorID,
		ResourceType: "material",
		ResourceID:   materialID,
		Category:     "materials",
	})
}

// =============================================================================
// Transactions (Goods Receipt / Issue / Scrap / Return)
// =============================================================================

// TransactionsAdapter satisfies the transactions package's ActivityLogger interface.
type TransactionsAdapter struct {
	svc *Service
}

func NewTransactionsAdapter(svc *Service) *TransactionsAdapter {
	return &TransactionsAdapter{svc: svc}
}

func (a *TransactionsAdapter) LogReceipt(ctx context.Context, actorID, txnID, txnNo, sku string, qty int) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Goods Receipt",
		Detail:       fmt.Sprintf("Penerimaan %s untuk %s sebanyak %d", txnNo, sku, qty),
		Type:         TypeSuccess,
		UserID:       actorID,
		ResourceType: "transaction",
		ResourceID:   txnID,
		Category:     "transactions",
	})
}

func (a *TransactionsAdapter) LogIssue(ctx context.Context, actorID, txnID, txnNo, sku, projectCode string, qty int) {
	detail := fmt.Sprintf("Pengeluaran %s untuk %s sebanyak %d", txnNo, sku, qty)
	if projectCode != "" {
		detail = fmt.Sprintf("Pengeluaran %s untuk %s sebanyak %d ke proyek %s", txnNo, sku, qty, projectCode)
	}
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Goods Issue",
		Detail:       detail,
		Type:         TypeInfo,
		UserID:       actorID,
		ResourceType: "transaction",
		ResourceID:   txnID,
		Category:     "transactions",
	})
}

func (a *TransactionsAdapter) LogScrapReturn(ctx context.Context, actorID, txnID, txnNo, txnType, sku string, qty int, reason string) {
	action := "Scrap Material"
	logType := TypeWarning
	if txnType == "return" {
		action = "Material Returned"
		logType = TypeInfo
	}
	a.svc.LogSafe(ctx, CreateInput{
		Action:       action,
		Detail:       fmt.Sprintf("%s untuk %s sebanyak %d (alasan: %s)", txnNo, sku, qty, reason),
		Type:         logType,
		UserID:       actorID,
		ResourceType: "transaction",
		ResourceID:   txnID,
		Category:     "transactions",
	})
}

// =============================================================================
// Tools
// =============================================================================

// ToolsAdapter satisfies the tools package's ActivityLogger interface.
type ToolsAdapter struct {
	svc *Service
}

func NewToolsAdapter(svc *Service) *ToolsAdapter {
	return &ToolsAdapter{svc: svc}
}

func (a *ToolsAdapter) LogToolCreated(ctx context.Context, actorID, toolID, sku, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Tool Created",
		Detail:       fmt.Sprintf("Alat %s (%s) ditambahkan", sku, name),
		Type:         TypeSuccess,
		UserID:       actorID,
		ResourceType: "tool",
		ResourceID:   toolID,
		Category:     "tools",
	})
}

func (a *ToolsAdapter) LogToolUpdated(ctx context.Context, actorID, toolID, sku, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Tool Updated",
		Detail:       fmt.Sprintf("Alat %s (%s) diperbarui", sku, name),
		Type:         TypeInfo,
		UserID:       actorID,
		ResourceType: "tool",
		ResourceID:   toolID,
		Category:     "tools",
	})
}

func (a *ToolsAdapter) LogToolDeleted(ctx context.Context, actorID, toolID, sku, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Tool Deleted",
		Detail:       fmt.Sprintf("Alat %s (%s) dihapus", sku, name),
		Type:         TypeWarning,
		UserID:       actorID,
		ResourceType: "tool",
		ResourceID:   toolID,
		Category:     "tools",
	})
}

func (a *ToolsAdapter) LogToolCheckout(ctx context.Context, actorID, toolID, sku, name, borrowerName string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Tool Checked Out",
		Detail:       fmt.Sprintf("%s (%s) dipinjam oleh %s", name, sku, borrowerName),
		Type:         TypeInfo,
		UserID:       actorID,
		ResourceType: "tool",
		ResourceID:   toolID,
		Category:     "tools",
	})
}

func (a *ToolsAdapter) LogToolReturn(ctx context.Context, actorID, toolID, sku, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Tool Returned",
		Detail:       fmt.Sprintf("%s (%s) telah dikembalikan", name, sku),
		Type:         TypeSuccess,
		UserID:       actorID,
		ResourceType: "tool",
		ResourceID:   toolID,
		Category:     "tools",
	})
}

func (a *ToolsAdapter) LogToolMaintenance(ctx context.Context, actorID, toolID, sku, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Tool Maintenance",
		Detail:       fmt.Sprintf("%s (%s) ditandai sedang maintenance", name, sku),
		Type:         TypeWarning,
		UserID:       actorID,
		ResourceType: "tool",
		ResourceID:   toolID,
		Category:     "tools",
	})
}

func (a *ToolsAdapter) LogToolAvailable(ctx context.Context, actorID, toolID, sku, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Tool Available",
		Detail:       fmt.Sprintf("%s (%s) ditandai tersedia kembali", name, sku),
		Type:         TypeSuccess,
		UserID:       actorID,
		ResourceType: "tool",
		ResourceID:   toolID,
		Category:     "tools",
	})
}

// =============================================================================
// Users (admin actions)
// =============================================================================

// UsersAdapter satisfies the users package's ActivityLogger interface.
type UsersAdapter struct {
	svc *Service
}

func NewUsersAdapter(svc *Service) *UsersAdapter {
	return &UsersAdapter{svc: svc}
}

func (a *UsersAdapter) LogUserCreated(ctx context.Context, actorID, targetID, email, role string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "User Created",
		Detail:       fmt.Sprintf("Pengguna %s (%s) ditambahkan", email, role),
		Type:         TypeSuccess,
		UserID:       actorID,
		ResourceType: "user",
		ResourceID:   targetID,
		Category:     "users",
	})
}

func (a *UsersAdapter) LogUserUpdated(ctx context.Context, actorID, targetID, email string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "User Updated",
		Detail:       fmt.Sprintf("Profil pengguna %s diperbarui", email),
		Type:         TypeInfo,
		UserID:       actorID,
		ResourceType: "user",
		ResourceID:   targetID,
		Category:     "users",
	})
}

func (a *UsersAdapter) LogUserStatusChanged(ctx context.Context, actorID, targetID, email, status string) {
	logType := TypeInfo
	if status == "inactive" {
		logType = TypeWarning
	}
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "User Status Changed",
		Detail:       fmt.Sprintf("Status pengguna %s diubah menjadi %s", email, status),
		Type:         logType,
		UserID:       actorID,
		ResourceType: "user",
		ResourceID:   targetID,
		Category:     "users",
	})
}

func (a *UsersAdapter) LogUserPasswordReset(ctx context.Context, actorID, targetID, email string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "User Password Reset",
		Detail:       fmt.Sprintf("Password pengguna %s direset oleh admin", email),
		Type:         TypeWarning,
		UserID:       actorID,
		ResourceType: "user",
		ResourceID:   targetID,
		Category:     "users",
	})
}

func (a *UsersAdapter) LogUserDeleted(ctx context.Context, actorID, targetID, email string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "User Deleted",
		Detail:       fmt.Sprintf("Pengguna %s dihapus", email),
		Type:         TypeDanger,
		UserID:       actorID,
		ResourceType: "user",
		ResourceID:   targetID,
		Category:     "users",
	})
}

// =============================================================================
// Vendors
// =============================================================================

// VendorsAdapter satisfies the vendors package's ActivityLogger interface.
type VendorsAdapter struct {
	svc *Service
}

func NewVendorsAdapter(svc *Service) *VendorsAdapter {
	return &VendorsAdapter{svc: svc}
}

func (a *VendorsAdapter) LogVendorCreated(ctx context.Context, actorID, vendorID, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Vendor Created",
		Detail:       fmt.Sprintf("Vendor %s ditambahkan", name),
		Type:         TypeSuccess,
		UserID:       actorID,
		ResourceType: "vendor",
		ResourceID:   vendorID,
		Category:     "vendors",
	})
}

func (a *VendorsAdapter) LogVendorUpdated(ctx context.Context, actorID, vendorID, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Vendor Updated",
		Detail:       fmt.Sprintf("Vendor %s diperbarui", name),
		Type:         TypeInfo,
		UserID:       actorID,
		ResourceType: "vendor",
		ResourceID:   vendorID,
		Category:     "vendors",
	})
}

func (a *VendorsAdapter) LogVendorDeleted(ctx context.Context, actorID, vendorID, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Vendor Deleted",
		Detail:       fmt.Sprintf("Vendor %s dihapus", name),
		Type:         TypeWarning,
		UserID:       actorID,
		ResourceType: "vendor",
		ResourceID:   vendorID,
		Category:     "vendors",
	})
}

// =============================================================================
// Projects
// =============================================================================

// ProjectsAdapter satisfies the projects package's ActivityLogger interface.
type ProjectsAdapter struct {
	svc *Service
}

func NewProjectsAdapter(svc *Service) *ProjectsAdapter {
	return &ProjectsAdapter{svc: svc}
}

func (a *ProjectsAdapter) LogProjectCreated(ctx context.Context, actorID, projectID, code, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Project Created",
		Detail:       fmt.Sprintf("Proyek %s (%s) dibuat", code, name),
		Type:         TypeSuccess,
		UserID:       actorID,
		ResourceType: "project",
		ResourceID:   projectID,
		Category:     "projects",
	})
}

func (a *ProjectsAdapter) LogProjectUpdated(ctx context.Context, actorID, projectID, code, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Project Updated",
		Detail:       fmt.Sprintf("Proyek %s (%s) diperbarui", code, name),
		Type:         TypeInfo,
		UserID:       actorID,
		ResourceType: "project",
		ResourceID:   projectID,
		Category:     "projects",
	})
}

func (a *ProjectsAdapter) LogProjectDeleted(ctx context.Context, actorID, projectID, code, name string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Project Deleted",
		Detail:       fmt.Sprintf("Proyek %s (%s) dihapus", code, name),
		Type:         TypeWarning,
		UserID:       actorID,
		ResourceType: "project",
		ResourceID:   projectID,
		Category:     "projects",
	})
}

// =============================================================================
// Purchase Orders
// =============================================================================

// PurchaseOrdersAdapter satisfies the purchaseorders package's ActivityLogger interface.
type PurchaseOrdersAdapter struct {
	svc *Service
}

func NewPurchaseOrdersAdapter(svc *Service) *PurchaseOrdersAdapter {
	return &PurchaseOrdersAdapter{svc: svc}
}

func (a *PurchaseOrdersAdapter) LogPOCreated(ctx context.Context, actorID, poID, poNumber, vendorName string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Purchase Order Created",
		Detail:       fmt.Sprintf("PO %s untuk vendor %s dibuat", poNumber, vendorName),
		Type:         TypeSuccess,
		UserID:       actorID,
		ResourceType: "purchase_order",
		ResourceID:   poID,
		Category:     "purchase-orders",
	})
}

func (a *PurchaseOrdersAdapter) LogPOUpdated(ctx context.Context, actorID, poID, poNumber, status string) {
	detail := fmt.Sprintf("PO %s diperbarui", poNumber)
	if status != "" {
		detail = fmt.Sprintf("PO %s diperbarui (status: %s)", poNumber, status)
	}
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Purchase Order Updated",
		Detail:       detail,
		Type:         TypeInfo,
		UserID:       actorID,
		ResourceType: "purchase_order",
		ResourceID:   poID,
		Category:     "purchase-orders",
	})
}

func (a *PurchaseOrdersAdapter) LogPODeleted(ctx context.Context, actorID, poID, poNumber string) {
	a.svc.LogSafe(ctx, CreateInput{
		Action:       "Purchase Order Deleted",
		Detail:       fmt.Sprintf("PO %s dihapus", poNumber),
		Type:         TypeWarning,
		UserID:       actorID,
		ResourceType: "purchase_order",
		ResourceID:   poID,
		Category:     "purchase-orders",
	})
}
