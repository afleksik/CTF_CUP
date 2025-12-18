package utils

func IsValidDrink(name string) bool {
	validDrinks := map[string]bool{
		"beer":      true,
		"wine":      true,
		"cocktail":  true,
		"whiskey":   true,
		"champagne": true,
	}

	return validDrinks[name]
}

func GetDrinkPrice(drinkName string) int {
	prices := map[string]int{
		"beer":      500,  // 500 roubles
		"wine":      1000, // 1000 roubles
		"cocktail":  1000, // 1000 roubles
		"whiskey":   1500, // 1500 roubles
		"champagne": 500,  // 500 roubles
	}

	if price, ok := prices[drinkName]; ok {
		return price
	}

	return 100
}
