package repository

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"homeessentials/backend/internal/model"
)

func TestOrderListFilter_orderTypeCustom(t *testing.T) {
	custom := model.OrderTypeCustom
	got := orderListFilter(OrderListFilter{OrderType: &custom})
	if got["orderType"] != model.OrderTypeCustom {
		t.Fatalf("custom filter = %#v", got)
	}
}

func TestOrderListFilter_orderTypeShopIncludesLegacy(t *testing.T) {
	shop := model.OrderTypeShop
	got := orderListFilter(OrderListFilter{OrderType: &shop})
	orRaw, ok := got["$or"]
	if !ok {
		t.Fatalf("expected $or for shop filter, got %#v", got)
	}
	orList, ok := orRaw.([]bson.M)
	if !ok {
		t.Fatalf("$or type = %T", orRaw)
	}
	if len(orList) != 3 {
		t.Fatalf("$or len = %d, want 3", len(orList))
	}
}

func TestOrderListFilter_statusAndType(t *testing.T) {
	status := model.OrderStatusCreated
	custom := model.OrderTypeCustom
	got := orderListFilter(OrderListFilter{Status: &status, OrderType: &custom})
	if got["status"] != model.OrderStatusCreated {
		t.Fatalf("status = %#v", got["status"])
	}
	if got["orderType"] != model.OrderTypeCustom {
		t.Fatalf("orderType = %#v", got["orderType"])
	}
}
