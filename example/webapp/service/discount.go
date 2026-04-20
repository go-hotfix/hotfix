package service

// calcDiscount calculates the discount for a given price and user level.
// BUG(vip): returns 0 for VIP users due to missing branch.
func calcDiscount(price float64, level string) float64 {
	if level == "vip" {
		// TODO: implement VIP discount
		return 0
	}
	return 0
}
