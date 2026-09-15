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

const productsCollection = "products"

var ErrNotFound = errors.New("product not found")

type ProductRepository struct {
	col *mongo.Collection
}

func NewProductRepository(db *mongo.Database) *ProductRepository {
	return &ProductRepository{col: db.Collection(productsCollection)}
}

func (r *ProductRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "category", Value: 1}, {Key: "active", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: -1}},
		},
	}

	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("create product indexes: %w", err)
	}
	return nil
}

type ListFilter struct {
	Category *model.Category
	Active   *bool
	Page     int
	PageSize int
}

func productListFilter(f ListFilter) bson.M {
	filter := bson.M{}
	if f.Category != nil {
		filter["category"] = *f.Category
	}
	if f.Active != nil {
		filter["active"] = *f.Active
	}
	return filter
}

func (r *ProductRepository) Create(ctx context.Context, p *model.Product) error {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.ID.IsZero() {
		p.ID = primitive.NewObjectID()
	}

	_, err := r.col.InsertOne(ctx, p)
	if err != nil {
		return fmt.Errorf("insert product: %w", err)
	}
	return nil
}

func (r *ProductRepository) Update(ctx context.Context, p *model.Product) error {
	p.UpdatedAt = time.Now().UTC()

	res, err := r.col.ReplaceOne(ctx, bson.M{"_id": p.ID}, p)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ProductRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Product, error) {
	var p model.Product
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find product: %w", err)
	}
	return &p, nil
}

func (r *ProductRepository) FindByIDs(ctx context.Context, ids []primitive.ObjectID) (map[primitive.ObjectID]model.Product, error) {
	if len(ids) == 0 {
		return map[primitive.ObjectID]model.Product{}, nil
	}

	cur, err := r.col.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, fmt.Errorf("find products by ids: %w", err)
	}
	defer cur.Close(ctx)

	out := make(map[primitive.ObjectID]model.Product, len(ids))
	for cur.Next(ctx) {
		var p model.Product
		if err := cur.Decode(&p); err != nil {
			return nil, fmt.Errorf("decode product: %w", err)
		}
		out[p.ID] = p
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return out, nil
}

func (r *ProductRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ProductRepository) List(ctx context.Context, f ListFilter) ([]model.Product, int64, error) {
	if f.Page < 1 {
		f.Page = pagination.DefaultPage
	}
	if f.PageSize < 1 {
		f.PageSize = pagination.DefaultPageSize
	}

	filter := productListFilter(f)

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64((f.Page - 1) * f.PageSize)).
		SetLimit(int64(f.PageSize))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer cur.Close(ctx)

	var products []model.Product
	if err := cur.All(ctx, &products); err != nil {
		return nil, 0, fmt.Errorf("decode products: %w", err)
	}
	if products == nil {
		products = []model.Product{}
	}
	return products, total, nil
}
