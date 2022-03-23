package faker

import (
	"fmt"
	"math"
	"reflect"
)

// Currency Codes | Source: https://en.wikipedia.org/wiki/ISO_4217
var currencies = []string{
	"AED", "AFN", "ALL", "AMD", "ANG", "AOA", "ARS", "AUD", "AWG",
	"AZN", "BAM", "BBD", "BDT", "BGN", "BHD", "BIF", "BMD", "BND",
	"BOB", "BOV", "BRL", "BSD", "BTN", "BWP", "BYN", "BZD", "CAD",
	"CDF", "CHE", "CHF", "CHW", "CLF", "CLP", "CNY", "COP", "COU",
	"CRC", "CUC", "CUP", "CVE", "CZK", "DJF", "DKK", "DOP", "DZD",
	"EGP", "ERN", "ETB", "EUR", "FJD", "FKP", "GBP", "GEL", "GHS",
	"GIP", "GMD", "GNF", "GTQ", "GYD", "HKD", "HNL", "HRK", "HTG",
	"HUF", "IDR", "ILS", "INR", "IQD", "IRR", "ISK", "JMD", "JOD",
	"JPY", "KES", "KGS", "KHR", "KMF", "KPW", "KRW", "KWD", "KYD",
	"KZT", "LAK", "LBP", "LKR", "LRD", "LSL", "LYD", "MAD", "MDL",
	"MGA", "MKD", "MMK", "MNT", "MOP", "MRU", "MUR", "MVR", "MWK",
	"MXN", "MXV", "MYR", "MZN", "NAD", "NGN", "NIO", "NOK", "NPR",
	"NZD", "OMR", "PAB", "PEN", "PGK", "PHP", "PKR", "PLN", "PYG",
	"QAR", "RON", "RSD", "RUB", "RWF", "SAR", "SBD", "SCR", "SDG",
	"SEK", "SGD", "SHP", "SLL", "SOS", "SRD", "SSP", "STN", "SVC",
	"SYP", "SZL", "THB", "TJS", "TMT", "TND", "TOP", "TRY", "TTD",
	"TWD", "TZS", "UAH", "UGX", "USD", "USN", "UYI", "UYU", "UYW",
	"UZS", "VES", "VND", "VUV", "WST", "XAF", "XAG", "XAU", "XBA",
	"XBB", "XBC", "XBD", "XCD", "XDR", "XOF", "XPD", "XPF", "XPT",
	"XSU", "XTS", "XUA", "XXX", "YER", "ZAR", "ZMW", "ZWL",
}

// Money provides an interface to generate a custom price with or without a random currency code
type Money interface {
	Currency(v reflect.Value) (interface{}, error)
	Amount(v reflect.Value) (interface{}, error)
	AmountWithCurrency(v reflect.Value) (interface{}, error)
}

// Price struct
type Price struct {
}

var pri Money

// GetPrice returns a new Money interface of Price struct
func GetPrice() Money {
	mu.Lock()
	defer mu.Unlock()

	if pri == nil {
		pri = &Price{}
	}
	return pri
}

// SetPrice sets custom Money
func SetPrice(p Money) {
	pri = p
}

func (p Price) currency() string {
	return randomElementFromSliceString(currencies)
}

// Currency returns a random currency from currencies
func (p Price) Currency(v reflect.Value) (interface{}, error) {
	return p.currency(), nil
}

// Currency get fake Currency (IDR, USD)
func Currency() string {
	return singleFakeData(CurrencyTag, func() interface{} {
		p := Price{}
		return p.currency()
	}).(string)
}

func (p Price) amount() float64 {
	return precision(rand.Float64()*math.Pow10(rand.Intn(8)), rand.Intn(2)+1)
}

// Amount returns a random floating price amount
// with a random precision of [1,2] up to (10**8 - 1)
func (p Price) Amount(v reflect.Value) (interface{}, error) {
	kind := v.Kind()
	val := p.amount()
	if kind == reflect.Float32 {
		v.Set(reflect.ValueOf(float32(val)))
		return float32(val), nil
	}
	v.Set(reflect.ValueOf(val))
	return val, nil
}

func (p Price) amountwithcurrency() string {
	val := p.amount()
	return fmt.Sprintf("%s %f", p.currency(), val)
}

// AmountWithCurrency combines both price and currency together
func (p Price) AmountWithCurrency(v reflect.Value) (interface{}, error) {
	return p.amountwithcurrency(), nil
}

// AmountWithCurrency get fake AmountWithCurrency  USD 49257.100
func AmountWithCurrency() string {
	return singleFakeData(AmountWithCurrencyTag, func() interface{} {
		p := Price{}
		return p.amountwithcurrency()
	}).(string)
}

// precision | a helper function to set precision of price
func precision(val float64, pre int) float64 {
	div := math.Pow10(pre)
	return float64(int64(val*div)) / div
}
