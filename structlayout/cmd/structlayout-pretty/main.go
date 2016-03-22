package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// FIXME move this type to a shared package

type Field struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Start     int64  `json:"start"`
	End       int64  `json:"end"`
	Size      int64  `json:"size"`
	Align     int64  `json:"align"`
	IsPadding bool   `json:"is_padding"`
}

func main() {
	log.SetFlags(0)
	var fields []Field
	if err := json.NewDecoder(os.Stdin).Decode(&fields); err != nil {
		log.Fatal(err)
	}
	if len(fields) == 0 {
		return
	}
	max := fields[len(fields)-1].End
	maxLength := len(fmt.Sprintf("%d", max))
	padding := strings.Repeat(" ", maxLength+2)
	format := fmt.Sprintf(" %%%dd ", maxLength)
	pos := int64(0)
	fmt.Println(padding + "+--------+")
	for _, field := range fields {
		name := field.Name + " " + field.Type
		if field.IsPadding {
			name = "padding"
		}
		// TODO calculate max size width
		fmt.Printf(format+"|        | <- %s\n", pos, name)
		fmt.Println(padding + "+--------+")

		if field.Size == 1 {
			pos++
			continue
		}

		if field.Size > 2 {
			fmt.Println(padding + "-........-")
			fmt.Println(padding + "+--------+")
		}
		pos += field.Size
		fmt.Printf(format+"|        |\n", pos-1)
		fmt.Println(padding + "+--------+")
	}
}
