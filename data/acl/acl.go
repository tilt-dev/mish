package acl

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/windmilleng/mish/data"
	"github.com/windmilleng/mish/logging"
)

type Ownable interface {
	Owner() data.UserID
}

type HasEmptyValue interface {
	// Many types in Windmill have a special Empty value that everyone can read but
	// no one can write.
	IsEmptyID() bool
}

type Checker struct {
	reader data.UserReader
}

func NewChecker(reader data.UserReader) Checker {
	return Checker{reader: reader}
}

func (c Checker) CanRead(ctx context.Context, o Ownable, userID data.UserID) bool {
	v, ok := o.(HasEmptyValue)
	if ok && v.IsEmptyID() {
		return true
	}

	ownerID := o.Owner()
	if ownerID == userID || ownerID == data.PublicID {
		return true
	}

	owner, err := c.reader.LookupByUserID(ctx, ownerID)
	if err != nil {
		if grpc.Code(err) != codes.NotFound {
			logging.With(ctx).Errorf("Checker#Lookup: %v", err)
		}
		return false
	}

	return owner.IsPublic
}

func (c Checker) CanWrite(ctx context.Context, o Ownable, userID data.UserID) bool {
	v, ok := o.(HasEmptyValue)
	if ok && v.IsEmptyID() {
		return false
	}
	return o.Owner() == userID
}

func (c Checker) CanReadPointerAtSnap(ctx context.Context, s data.PointerAtSnapshot, userID data.UserID) bool {
	return c.CanRead(ctx, s.ID, userID) && c.CanRead(ctx, s.SnapID, userID)
}
