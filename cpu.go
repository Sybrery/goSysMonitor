package main

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

func PrintCpu() {
	c, err := cpu.Percent(time.Second, false)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	ci, err := cpu.Info()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if len(ci) == 0 {
		return
	}

	modelName := fmt.Sprintf("%v", ci[0].ModelName)

	fmt.Println(modelName, "\n", c)
}
