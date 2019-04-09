package repositories

// Inventory stored in DB
type Inventory struct {
	ProductID  int64
	StockCount int64
	Version    int64
}
