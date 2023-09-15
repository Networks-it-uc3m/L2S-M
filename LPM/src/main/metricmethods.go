package main

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

/** The purpose of this file is to save all the measuring methods, that will be set as arguments in the AddMetrics function
**/
func measureRtt(neighborIP string) float64 {

	log.Infof("Measuring rtt")

	out, err := exec.Command("ping", neighborIP, "-c", "10", "-q").Output()
	if err != nil {
		log.Errorf("Could not measure Rtt. Ping responds: %v", err)
		return 0
	}
	//fmt.Println(string(out[:]))

	// Regular expression pattern
	pattern := `rtt min/avg/max/mdev = [0-9.]+/([0-9.]+)/`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Find the first match in the input string
	match := re.FindStringSubmatch(string(out[:]))

	if len(match) < 2 {
		log.Errorf("Could not measure Rtt. Check the connection between the two nodes.")
		return 0
	}
	// Print the result
	log.Infof("Rtt between two links: %s", match[1])

	rtt, _ := strconv.ParseFloat(match[1], 64)
	return rtt
}

func measureJitter(neighborIP string) float64 {

	log.Infof("Measuring jitter")

	out, err := exec.Command("iperf3", "-u", "-p", "5202", "-c", neighborIP).Output()
	if err != nil {
		log.Errorf("Could not measure Jitter. %v", err)
		return 0
	}
	//fmt.Println(string(out[:]))

	// Define the regular expression patterns
	senderPattern := "receiver"
	msPattern := "[0-9.]+ ms"

	// Compile the regular expressions
	senderRegex := regexp.MustCompile(senderPattern)
	msRegex := regexp.MustCompile(msPattern)

	lines := strings.Split(string(out[:]), "\n")

	for _, line := range lines {
		if senderRegex.MatchString(line) {
			// If the line contains "receiver", search for the 'ms' pattern
			msMatch := msRegex.FindStringSubmatch(line)

			// Loop through the 'ms' matches and print them

			log.Infof("Jitter between two links: %s", msMatch)

			jitter, err := strconv.ParseFloat(strings.TrimSuffix(msMatch[0], " ms"), 64)
			if err != nil {
				log.Errorf("Could not parse the jitter metric. %v", err)
			}
			return jitter

		}
	}
	// Find the first match in the input string
	//match := re.FindStringSubmatch(string(out[:]))

	// Print the result
	//fmt.Println("Extracted number:", match)

	//netDat.Rtt, err = strconv.ParseFloat(match[0], 0)
	return 0
}

func measureThroughput(neighborIP string) float64 {
	log.Infof("Measuring throughput")

	out, err := exec.Command("iperf3", "-c", neighborIP).Output()
	if err != nil {
		log.Errorf("Could not measure Throughput. %v", err)
		return 0
	}
	//fmt.Println(string(out[:]))

	// Define the regular expression patterns
	senderPattern := "receiver"
	msPattern := "[0-9.]+ Gbits/sec"

	// Compile the regular expressions
	senderRegex := regexp.MustCompile(senderPattern)
	msRegex := regexp.MustCompile(msPattern)

	lines := strings.Split(string(out[:]), "\n")

	for _, line := range lines {
		if senderRegex.MatchString(line) {
			// If the line contains "receiver", search for the 'ms' pattern
			msMatch := msRegex.FindStringSubmatch(line)

			log.Infof("Throughput between two links: %s", msMatch)
			throughput, err := strconv.ParseFloat(strings.TrimSuffix(msMatch[0], " Gbits/sec"), 64)
			if err != nil {
				log.Errorf("Could not parse the throughput metric. %v", err)
			}
			return throughput
		}

	}
	return 0

}

//////////////////////////////////////////////////////////
//////////////// SERVER METHODS //////////////////////////
//////////////////////////////////////////////////////////

func iperfTCP() {
	exec.Command("iperf3", "-s", "-p", "5201").Run()
}

func iperfUDP() {
	exec.Command("iperf3", "-s", "-p", "5202").Run()
}
