package main

import "encoding/binary"

func uint64ToBytes(i uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], i)
	return buf[:]
}

func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func indexOfString(s []string, q string) int {
	for i, v := range s {
		if v == q {
			return i
		}
	}

	return -1
}
