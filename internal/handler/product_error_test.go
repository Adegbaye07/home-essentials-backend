package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"homeessentials/backend/internal/controller"
	"homeessentials/backend/internal/model"
	"homeessentials/backend/internal/pricing"
	"homeessentials/backend/internal/repository"
)

func TestWriteProductError_productDeleteConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantError  string
	}{
		{
			name:       "pending order",
			err:        controller.ErrProductDeletePendingOrder,
			wantStatus: http.StatusConflict,
			wantCode:   "product_delete_pending_order",
			wantError:  "There is a pending order for this item.",
		},
		{
			name:       "incomplete order",
			err:        controller.ErrProductDeleteIncompleteOrder,
			wantStatus: http.StatusConflict,
			wantCode:   "product_delete_incomplete_order",
			wantError:  "There is an incomplete order for this item.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			writeProductError(c, tt.err)

			if w.Code != tt.wantStatus {
				t.Fatalf("status %d want %d body %s", w.Code, tt.wantStatus, w.Body.String())
			}
			var body struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error != tt.wantError {
				t.Fatalf("error %q want %q", body.Error, tt.wantError)
			}
			if body.Code != tt.wantCode {
				t.Fatalf("code %q want %q", body.Code, tt.wantCode)
			}
		})
	}
}

func TestWriteProductError_notFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	writeProductError(c, repository.ErrNotFound)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", w.Code)
	}
}

func TestWriteProductError_tierValidationReadable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	maxOne := 1
	tierErr := pricing.ValidateTiers(model.SizeM, []model.QtyTier{
		{MinQty: 1, MaxQty: &maxOne, UnitPriceKobo: 100},
		{MinQty: 3, UnitPriceKobo: 90},
	})
	writeProductError(c, tierErr)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400 body %s", w.Code, w.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.Error, "second price tier for size M") {
		t.Fatalf("error %q not readable", body.Error)
	}
}
