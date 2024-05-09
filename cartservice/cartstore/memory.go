package cartstore

import (
	"context"
	"sync"

	pb "cartservice/proto"
)

// Data is stored in memory using a nested map structure.
type memoryCartStore struct {
	// Read-write lock
	sync.RWMutex
	carts map[string]map[string]int32
}

// Add Item
func (s *memoryCartStore) AddItem(ctx context.Context, userID, productID string, quantity int32, out *pb.Empty) (r *pb.Empty, err error) {
	s.Lock()
	defer s.Unlock()

	if cart, ok := s.carts[userID]; ok {
		if currentQuantity, ok := cart[productID]; ok {
			cart[productID] = currentQuantity + quantity
		} else {
			cart[productID] = quantity
		}
		s.carts[userID] = cart
	} else {
		s.carts[userID] = map[string]int32{productID: quantity}
	}
	return out, nil
}

// Empty Cart
func (s *memoryCartStore) EmptyCart(ctx context.Context, userID string) (out *pb.Empty, err error) {
	s.Lock()
	defer s.Unlock()
	out = new(pb.Empty)
	delete(s.carts, userID)
	return out, nil
}

// Get Cart
func (s *memoryCartStore) GetCart(ctx context.Context, userID string) (*pb.Cart, error) {
	s.RLock()
	defer s.RUnlock()

	if cart, ok := s.carts[userID]; ok {
		items := make([]*pb.CartItem, 0, len(cart))
		for p, q := range cart {
			items = append(items, &pb.CartItem{ProductId: p, Quantity: q})
		}
		return &pb.Cart{UserId: userID, Items: items}, nil
	}
	return &pb.Cart{UserId: userID}, nil
}
