package main

import (
	"fmt"

)

func main() {
	mkl := make(map[int]map[int]int)
	mkn := make(map[int]int)
	mkn[1] =1
	mkl[1] = mkn
	for _,val := range mkl{
		val[1]=2
	}
	fmt.Printf("res : %d", mkl[1][1])
}
