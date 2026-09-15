package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"homeessentials/backend/internal/model"
	"homeessentials/backend/internal/pagination"
)

const ordersCollection = "orders"

var ErrOrderNotFound = errors.New("order not found")

type OrderRepository struct {
	col *mongo.Collection
}

func NewOrderRepository(db *mongo.Database) *OrderRepository {
	return &OrderRepository{col: db.Collection(ordersCollection)}
}

func (r *OrderRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "paystackReference", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}},
		},
		{
			Keys:    bson.D{{Key: "trackingNumber", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys: bson.D{{Key: "items.productId", Value: 1}, {Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "abandonedAt", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "orderType", Value: 1}, {Key: "createdAt", Value: -1}},
		},
	}

	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("create order indexes: %w", err)
	}
	return nil
}

type OrderListFilter struct {
	Status    *model.OrderStatus
	OrderType *model.OrderType
	Page      int
	PageSize  int
}

func orderListFilter(f OrderListFilter) bson.M {
	filter := bson.M{}
	if f.Status != nil {
		filter["status"] = *f.Status
	}
	if f.OrderType != nil {
		t := f.OrderType.Normalized()
		if t == model.OrderTypeShop {
			// Legacy shop documents may omit orderType.
			filter["$or"] = []bson.M{
				{"orderType": model.OrderTypeShop},
				{"orderType": bson.M{"$exists": false}},
				{"orderType": ""},
			}
		} else {
			filter["orderType"] = t
		}
	}
	return filter
}

func (r *OrderRepository) Create(ctx context.Context, o *model.Order) error {
	now := time.Now().UTC()
	o.CreatedAt = now
	o.UpdatedAt = now
	if o.ID.IsZero() {
		o.ID = primitive.NewObjectID()
	}

	_, err := r.col.InsertOne(ctx, o)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Order, error) {
	var o model.Order
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&o)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("find order: %w", err)
	}
	return &o, nil
}

func (r *OrderRepository) List(ctx context.Context, f OrderListFilter) ([]model.Order, int64, error) {
	if f.Page < 1 {
		f.Page = pagination.DefaultPage
	}
	if f.PageSize < 1 {
		f.PageSize = pagination.DefaultPageSize
	}

	filter := orderListFilter(f)

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64((f.Page - 1) * f.PageSize)).
		SetLimit(int64(f.PageSize))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer cur.Close(ctx)

	var orders []model.Order
	if err := cur.All(ctx, &orders); err != nil {
		return nil, 0, fmt.Errorf("decode orders: %w", err)
	}
	if orders == nil {
		orders = []model.Order{}
	}
	return orders, total, nil
}

func (r *OrderRepository) Update(ctx context.Context, o *model.Order) error {
	o.UpdatedAt = time.Now().UTC()

	res, err := r.col.ReplaceOne(ctx, bson.M{"_id": o.ID}, o)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (r *OrderRepository) FindByPaystackReference(ctx context.Context, ref string) (*model.Order, error) {
	var o model.Order
	err := r.col.FindOne(ctx, bson.M{"paystackReference": ref}).Decode(&o)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("find order by reference: %w", err)
	}
	return &o, nil
}

// ProductDeleteBlockReason reports whether non-delivered orders reference the product.
// Pending takes precedence over incomplete when both exist.
func (r *OrderRepository) ProductDeleteBlockReason(ctx context.Context, productID primitive.ObjectID) (blockPending, blockIncomplete bool, err error) {
	err = r.col.FindOne(ctx, bson.M{
		"items.productId": productID,
		"status":          model.OrderStatusPendingPayment,
	}).Err()
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return false, false, fmt.Errorf("find pending order for product: %w", err)
		}
	} else {
		blockPending = true
	}

	err = r.col.FindOne(ctx, bson.M{
		"items.productId": productID,
		"status": bson.M{"$nin": []model.OrderStatus{
			model.OrderStatusDelivered,
			model.OrderStatusPendingPayment,
			model.OrderStatusAbandoned,
		}},
	}).Err()
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return blockPending, false, fmt.Errorf("find incomplete order for product: %w", err)
		}
		return blockPending, false, nil
	}
	return blockPending, true, nil
}

func (r *OrderRepository) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete order: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (r *OrderRepository) FindForTrack(ctx context.Context, tracking, email string) (*model.Order, error) {
	var o model.Order
	err := r.col.FindOne(ctx, bson.M{
		"trackingNumber": tracking,
		"customer.email": email,
	}).Decode(&o)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("find order for track: %w", err)
	}
	return &o, nil
}

// DeleteAbandonedBefore hard-deletes shop abandoned orders with abandonedAt strictly before cutoff.
// Custom (recreate) orders are never deleted by this cleanup.
func (r *OrderRepository) DeleteAbandonedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	cutoff = cutoff.UTC()
	res, err := r.col.DeleteMany(ctx, bson.M{
		"status":      model.OrderStatusAbandoned,
		"abandonedAt": bson.M{"$lt": cutoff},
		"orderType":   bson.M{"$ne": model.OrderTypeCustom},
	})
	if err != nil {
		return 0, fmt.Errorf("delete abandoned orders: %w", err)
	}
	return res.DeletedCount, nil
}
