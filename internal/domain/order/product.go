package order

type Product struct {
	// Id            int64
	// OrderId       int64
	ProductId            int64
	ProductName          string
	ProductAmountInCore  int32
	ProductAmountInOrder int32
	ProductPrice         int32
	ItemExists           bool
}
