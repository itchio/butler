package uploader

import (
	"log"
	"os"
	"strconv"
)

func fromEnv(envName string, defaultValue int) int {
	v := os.Getenv(envName)
	if v != "" {
		iv, err := strconv.Atoi(v)
		if err == nil {
			log.Printf("Override set: %s = %d", envName, iv)
			return iv
		}
	}
	return defaultValue
}
