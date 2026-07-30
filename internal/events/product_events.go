package events

type ProductCreated struct {
	ProductID int64
	Name      string
	Price     int64
	Amount    int64
}

func (e *ProductCreated) GetType() string {
	return "product.created"
}

type ProductDeleted struct {
	ProductID int64
}

func (e *ProductDeleted) GetType() string {
	return "product.deleted"
}
