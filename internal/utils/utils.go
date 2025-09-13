package utils

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	PbsJobIdRegex = regexp.MustCompile(`pbs_jobs\.service\/(?:jobid|jobs)\/(\d+(?:\[\d+\])?)(?:\.(\d+))?`)
)

func BooleanToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func GetCgroupJobId(cgroupPath string) string {
	var jobId, jobIndex string

	matches := PbsJobIdRegex.FindStringSubmatch(cgroupPath)

	if len(matches) > 1 {
		jobId = matches[1]
	}
	if len(matches) > 2 {
		jobIndex = matches[2]
	}

	if jobId != "" && jobIndex != "" {
		if index, err := strconv.ParseInt(jobIndex, 10, 32); err == nil && index > 1 {
			jobId = fmt.Sprintf("%s[%d]", jobId, index-1)
		}
	}

	return jobId
}

func MustHostname() string {
	h, err := os.Hostname()
	if err != nil {
		panic(fmt.Sprintf("Failed to get hostname: %v", err))
	}

	return h
}

// The List Format for cpus and mems is a comma-separated list of CPU
// or memory-node numbers and ranges of numbers, in ASCII decimal.
func ParseListFormat(listFormat string) ([]int, error) {
	listFormat = strings.TrimSpace(listFormat)
	if listFormat == "" {
		return nil, nil
	}

	var result []int
	for part := range strings.SplitSeq(listFormat, ",") {
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			start, _ := strconv.Atoi(rangeParts[0])
			end, _ := strconv.Atoi(rangeParts[1])
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
		} else {
			cpu, _ := strconv.Atoi(part)
			result = append(result, cpu)
		}
	}

	return result, nil
}

func ParseWalltime(walltime string) int64 {
	walltimeSeconds := int64(0)
	if walltime == "" {
		return 0
	}

	walltimeParts := strings.Split(walltime, ":")
	hours, _ := strconv.ParseInt(walltimeParts[0], 10, 0)
	minutes, _ := strconv.ParseInt(walltimeParts[1], 10, 0)
	seconds, _ := strconv.ParseInt(walltimeParts[2], 10, 0)

	walltimeSeconds += 3600 * hours
	walltimeSeconds += 60 * minutes
	walltimeSeconds += seconds

	return walltimeSeconds
}

func ReadFileSingleLine(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}

func DirectoryExists(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}
