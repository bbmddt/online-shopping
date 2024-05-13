package handler

import (
	"bytes"
	"context"
	"log"

	pb "emailservice/proto"
)

// send email
type DummyEmailService struct{}

// Log
var (
	buf    bytes.Buffer
	logger = log.New(&buf, "logger: ", log.Lshortfile)
)

func (s *DummyEmailService) SendOrderConfirmation(ctx context.Context, in *pb.SendOrderConfirmationRequest) (out *pb.Empty, e error) {
	logger.Printf("The email has been sent to: %s .", in.Email)
	out = new(pb.Empty)
	return out, nil
}
