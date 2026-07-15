package events

type ProductCreated struct {
	ProductID int
	Name      string
	Price     int
}

func (e ProductCreated) GetType() string {
	return "product.created"
}

type ProductDeleted struct {
	ProductID int
}

func (e ProductDeleted) GetType() string {
	return "product.deleted"
}
