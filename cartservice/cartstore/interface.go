package cartstore

import (
	"context"

	pb "cartservice/proto"
)

// Cart interface
type CartStore interface {
	AddItem(ctx context.Context, userID, productID string, quantity int32, out *pb.Empty) (r *pb.Empty, err error)
	EmptyCart(ctx context.Context, userID string) (*pb.Empty, error)
	GetCart(ctx context.Context, userID string) (*pb.Cart, error)
}

// Instantiate CartStore
func NewMemoryCartStore() CartStore {
	return &memoryCartStore{
		carts: make(map[string]map[string]int32),
	}
}
