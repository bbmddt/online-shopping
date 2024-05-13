package handler

import (
	"fmt"
	"math/rand"
	"time"
)

var localRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// CreateTrackingId generates a tracking ID for simulation
func CreateTrackingId(salt string) string {
	return fmt.Sprintf("%c%c-%d%s-%d%s",
		getRandomLetterCode(),
		getRandomLetterCode(),
		len(salt),
		getRandomNumber(3),
		len(salt)/2,
		getRandomNumber(7),
	)
}

// generates a random letter code
func getRandomLetterCode() uint32 {
	return 65 + uint32(localRand.Intn(25))
}

// generates a random number with specified digits
func getRandomNumber(digits int) string {
	str := ""
	for i := 0; i < digits; i++ {
		str = fmt.Sprintf("%s%d", str, localRand.Intn(10))
	}

	return str
}
