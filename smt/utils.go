package smt

import (
	"math/big"
)

func hasBit(data []byte, position int) int {
	if int(data[position/8])&(1<<(uint(position)%8)) > 0 {
		return 1
	}
	return 0
}

func isLeft(key []byte, depth int, height int) bool {
	keyInt := new(big.Int).SetBytes(key)
	leftShifted := keyInt.Lsh(keyInt, uint(depth))
	rightShifted := leftShifted.Rsh(leftShifted, uint(height)-2)
	return rightShifted.Mod(rightShifted, big.NewInt(2)).Cmp(big.NewInt(0)) == 0
}

func setBit(data []byte, position int) {
	n := int(data[position/8])
	n |= (1 << (uint(position) % 8))
	data[position/8] = byte(n)
}

func countSetBits(data []byte) int {
	count := 0
	for i := 0; i < len(data)*8; i++ {
		if hasBit(data, i) == 1 {
			count++
		}
	}
	return count
}

func emptyBytes(length int) []byte {
	b := make([]byte, length)
	return b
}
