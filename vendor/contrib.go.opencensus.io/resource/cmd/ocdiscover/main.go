package main

import (
	"context"
	"fmt"
	"log"

	"contrib.go.opencensus.io/resource/auto"
	"go.opencensus.io/resource"
)

func main() {
	res, err := auto.Detect(context.Background())
	if err != nil {
		log.Fatalf("detecting resource info failed: %s", err)
	}
	fmt.Printf("%s='%s' %s='%s'\n", resource.EnvVarType, res.Type, resource.EnvVarLabels, resource.EncodeLabels(res.Labels))
}
