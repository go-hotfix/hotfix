package service

import (
	"fmt"
	"net/http"
	"strconv"
)

// OrderHandler handles /order requests.
// Query params: price (float), level (string: "normal" or "vip")
func OrderHandler(w http.ResponseWriter, r *http.Request) {
	priceStr := r.URL.Query().Get("price")
	level := r.URL.Query().Get("level")
	if level == "" {
		level = "normal"
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		http.Error(w, "invalid price", http.StatusBadRequest)
		return
	}

	discount := calcDiscount(price, level)
	final := price - discount

	fmt.Fprintf(w, "price: %.0f, discount: %.0f, final: %.0f\n", price, discount, final)
}
