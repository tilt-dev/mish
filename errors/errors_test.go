package errors

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestPreserveContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ctx.Err()
	if !IsCanceled(err) {
		t.Errorf("Expected canceled: %v", err)
	}

	err = Propagatef(err, "Msg1")
	if !IsCanceled(err) {
		t.Errorf("Expected canceled: %v", err)
	}

	err = Propagatef(err, "Msg2(%s)", "hi")
	if !IsCanceled(err) {
		t.Errorf("Expected canceled: %v", err)
	}
}

func TestPreserveGRPCCanceled(t *testing.T) {
	err := grpc.Errorf(codes.Canceled, "Canceled")
	if !IsCanceled(err) {
		t.Errorf("Expected canceled: %v", err)
	}

	err = Propagatef(err, "Msg1")
	if !IsCanceled(err) {
		t.Errorf("Expected canceled: %v", err)
	}
}
