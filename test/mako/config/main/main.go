package main

import (
	yaml "gopkg.in/yaml.v2"
	"knative.dev/pkg/test/mako/config"

	"fmt"
)

func main() {
	channel1 := config.Channel{Name: "test1", Identity: "fdsfasdfsf"}
	channel2 := config.Channel{Name: "test2", Identity: "fdsfdsfdsf"}
	mp := make(map[string][]config.Channel, 0)
	mp["BC1"] = []config.Channel{channel1}
	mp["BC2"] = []config.Channel{channel1, channel2}
	res, _ := yaml.Marshal(config.SlackConfig{BenchmarkChannels: mp})
	fmt.Println(string(res))

	// channel := config.Channel{Name: "test1", Identity: "fdsfdsfdsf"}
	// res, _ := yaml.Marshal(channel)
	// fmt.Println(string(res))

	// 	res := `
	// name: test1
	// identity: fdsfasdfsf`
	// 	newConfig := &config.Channel{}
	// 	err := yaml.Unmarshal([]byte(res), newConfig)
	// 	fmt.Println("error is: %v", err)
	// 	fmt.Printf("%v", *newConfig)
}
