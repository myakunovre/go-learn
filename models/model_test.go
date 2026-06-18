package models

import "testing"

func TestProduct_Struct(t *testing.T) {
	product := Product{
		ID:    1,
		Name:  "test",
		Price: 1,
	}

	if product.ID != 1 {
		t.Errorf("id = %d, expected 1", product.ID)
	}

	if product.Name != "test" {
		t.Errorf("name = %s, expected test", product.Name)
	}

	if product.Price != 1 {
		t.Errorf("price = %d, expected 1", product.Price)
	}
}
