package main

import (
	"os"
	"time"

	"github.com/Networks-it-uc3m/LPM/internal/swmintegration"
)

func main() {

	swmintegration.RunExporter(time.Minute*5, os.Getenv("TOPOLOGY_NAMESPACE"))

}
