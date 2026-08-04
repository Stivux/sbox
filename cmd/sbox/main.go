package main

import (
	"fmt"

	"github.com/sagernet/sing-box/option"
)

func main() {
	// You can now use the option structs
	var outbound option.V2RayXHTTPOptions
	// ... your code using the option package
	fmt.Println("Using sing-box-lx packages", outbound)
}
