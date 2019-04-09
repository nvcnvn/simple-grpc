package repositories

// InventoryQuantityUpdateError tell which item cannot update and reason
type InventoryQuantityUpdateError interface {
	error
	ProductID() int64
}
