package repositories

// Inventory store in DB
type Inventory struct {
	ID         uint
	StockCount uint
	Version    uint
}
