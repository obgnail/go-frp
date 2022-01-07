package utils

import (
	"fmt"
	"testing"
)

func TestAes(t *testing.T) {
	plaintext := []byte("我 爱 你")
	fmt.Println("明文:", string(plaintext))
	ciptext := AesEny(plaintext)
	fmt.Println("加密:", ciptext)
	platext2 := AesDec(ciptext)
	fmt.Println("解密:", string(platext2))
}
