package repositories

// Inventory stored in DB
type Inventory struct {
	ProductID  int64
	StockCount int64
	Version    int64
}

// Order not stored in DB.. for now
type Order struct {
	ProductID int64
	Quantity  int64
}
