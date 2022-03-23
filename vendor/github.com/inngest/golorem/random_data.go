// Package randomdata implements a bunch of simple ways to generate (pseudo) random data
package lorem

const (
	Male         int = 0
	Female       int = 1
	RandomGender int = 2
)

// Returns a random part of a slice
func (self *Lorem) randomFrom(source []string) string {
	return source[self.r.Intn(len(source))]
}

// Returns a random first name, gender decides the gender of the name
func (self *Lorem) FirstName(gender int) string {
	var name = ""
	switch gender {
	case Male:
		name = self.randomFrom(firstNamesMale)
		break
	case Female:
		name = self.randomFrom(firstNamesFemale)
		break
	default:
		name = self.FirstName(self.r.Intn(2))
		break
	}
	return name
}

// Returns a random last name
func (self *Lorem) LastName() string {
	return self.randomFrom(lastNames)
}

// Returns a combinaton of FirstName LastName randomized, gender decides the gender of the name
func (self *Lorem) FullName(gender int) string {
	return self.FirstName(gender) + " " + self.LastName()
}
