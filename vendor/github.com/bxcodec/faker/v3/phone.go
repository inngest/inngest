package faker

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bxcodec/faker/v3/support/slice"
)

var phone Phoner

// GetPhoner serves as a constructor for Phoner interface
func GetPhoner() Phoner {
	mu.Lock()
	defer mu.Unlock()

	if phone == nil {
		phone = &Phone{}
	}
	return phone
}

// SetPhoner sets custom Phoner
func SetPhoner(p Phoner) {
	phone = p
}

// Phoner serves overall tele-phonic contact generator
type Phoner interface {
	PhoneNumber(v reflect.Value) (interface{}, error)
	TollFreePhoneNumber(v reflect.Value) (interface{}, error)
	E164PhoneNumber(v reflect.Value) (interface{}, error)
}

// Phone struct
type Phone struct {
}

func (p Phone) phonenumber() string {
	randInt, _ := RandomInt(1, 10)
	str := strings.Join(slice.IntToString(randInt), "")
	return fmt.Sprintf("%s-%s-%s", str[:3], str[3:6], str[6:10])
}

// PhoneNumber generates phone numbers of type: "201-886-0269"
func (p Phone) PhoneNumber(v reflect.Value) (interface{}, error) {
	return p.phonenumber(), nil
}

// Phonenumber get fake phone number
func Phonenumber() string {
	return singleFakeData(PhoneNumber, func() interface{} {
		p := Phone{}
		return p.phonenumber()
	}).(string)
}

func (p Phone) tollfreephonenumber() string {
	out := ""
	boxDigitsStart := []string{"777", "888"}

	ints, _ := RandomInt(1, 9)
	for index, v := range slice.IntToString(ints) {
		if index == 3 {
			out += "-"
		}
		out += v
	}
	return fmt.Sprintf("(%s) %s", boxDigitsStart[rand.Intn(1)], out)
}

// TollFreePhoneNumber generates phone numbers of type: "(888) 937-7238"
func (p Phone) TollFreePhoneNumber(v reflect.Value) (interface{}, error) {
	return p.tollfreephonenumber(), nil
}

// TollFreePhoneNumber get fake TollFreePhoneNumber
func TollFreePhoneNumber() string {
	return singleFakeData(TollFreeNumber, func() interface{} {
		p := Phone{}
		return p.tollfreephonenumber()
	}).(string)
}

func (p Phone) e164PhoneNumber() string {
	out := ""
	boxDigitsStart := []string{"7", "8"}
	ints, _ := RandomInt(1, 10)

	for _, v := range slice.IntToString(ints) {
		out += v
	}
	return fmt.Sprintf("+%s%s", boxDigitsStart[rand.Intn(1)], strings.Join(slice.IntToString(ints), ""))
}

// E164PhoneNumber generates phone numbers of type: "+27113456789"
func (p Phone) E164PhoneNumber(v reflect.Value) (interface{}, error) {
	return p.e164PhoneNumber(), nil
}

// E164PhoneNumber get fake E164PhoneNumber
func E164PhoneNumber() string {
	return singleFakeData(E164PhoneNumberTag, func() interface{} {
		p := Phone{}
		return p.e164PhoneNumber()
	}).(string)
}
