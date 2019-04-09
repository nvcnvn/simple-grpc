package repositories

// Inventory store in DB
type Inventory struct {
	ID         int64
	StockCount int64
	Version    int64
}
