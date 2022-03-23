package faker

import (
	"reflect"
)

var address Addresser

// GetAddress returns a new Addresser interface of Address
func GetAddress() Addresser {
	mu.Lock()
	defer mu.Unlock()

	if address == nil {
		address = &Address{}
	}
	return address
}

// SetAddress sets custom Address
func SetAddress(net Addresser) {
	address = net
}

// Addresser is logical layer for Address
type Addresser interface {
	Latitude(v reflect.Value) (interface{}, error)
	Longitude(v reflect.Value) (interface{}, error)
}

// Address struct
type Address struct{}

func (i Address) latitude() float32 {
	return (rand.Float32() * 180) - 90
}

// Latitude sets latitude of the address
func (i Address) Latitude(v reflect.Value) (interface{}, error) {
	kind := v.Kind()
	val := i.latitude()
	if kind == reflect.Float32 {
		return val, nil
	}
	return float64(val), nil
}

func (i Address) longitude() float32 {
	return (rand.Float32() * 360) - 180
}

// Longitude sets longitude of the address
func (i Address) Longitude(v reflect.Value) (interface{}, error) {
	kind := v.Kind()
	val := i.longitude()
	if kind == reflect.Float32 {
		return val, nil
	}
	return float64(val), nil
}

// Longitude get fake longitude randomly
func Longitude() float64 {
	return singleFakeData(LONGITUDE, func() interface{} {
		address := Address{}
		return float64(address.longitude())
	}).(float64)
}

// Latitude get fake latitude randomly
func Latitude() float64 {
	return singleFakeData(LATITUDE, func() interface{} {
		address := Address{}
		return float64(address.latitude())
	}).(float64)
}
