package main

// Measure the variation in gaps between SAFE vault names
// when using various different naming strategies.

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"time"
)

// Parameters

const totalNames int = 100
const namingStrategy = "uniform" // uniform, random, bestfit, quietesthalf
const spacingStrategy = "xordistance" // linear, xordistance

// Sorters

type ByName []uint64

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i] < a[j] }

// Functions

func main() {
	// set up random numbers
	nowNanos := time.Now().UnixNano()
	rand.Seed(nowNanos)
	fmt.Println("Seed is", nowNanos)
	// create names
	names := []uint64{}
	for i := 0; i < totalNames; i++ {
		// get name that suits the naming strategy
		var name uint64
		if namingStrategy == "uniform" {
			progress := float64(i) / float64(totalNames)
			name = uint64(float64(math.MaxUint64) * progress)
		} else if namingStrategy == "random" {
			name = rand.Uint64()
		} else if namingStrategy == "bestfit" {
			name = nameForBestFit(names)
		} else if namingStrategy == "quietesthalf" {
			name = nameForQuiestestHalf(names)
		} else {
			panic("Invalid naming strategy")
		}
		// add new name to the section
		names = append(names, name)
	}
	// get distances
	distances := []uint64{}
	// distances from 0 to first name
	firstDistance := names[0] - 0
	distances = append(distances, firstDistance)
	// all other distances between names
	for i, _ := range names {
		if i == 0 {
			continue
		}
		distance := names[i] - names[i-1]
		distances = append(distances, distance)
	}
	// distance from last name to MaxUint64
	lastName := names[len(names)-1]
	lastDistance := math.MaxUint64 - lastName
	distances = append(distances, lastDistance)
	// report
	sort.Sort(ByName(names))
	fmt.Println("\nNames (base32):")
	for _, n := range names {
		fmt.Println(nameStr(n))
	}
	fmt.Println("\nStandard deviation of distances:")
	fmt.Println(standardDeviation(distances))
}

func nameForBestFit(names []uint64) uint64 {
	name := rand.Uint64()
	// if this is the first node
	// or names are random (ie the largest space is not being targeted)
	// add it now
	if len(names) == 0 {
		return name
	}
	// get the maximum spacing between existing names
	var maxSpacing uint64
	var minName uint64
	var maxName uint64
	sort.Sort(ByName(names))
	// find the maximum space between names
	for i, _ := range names {
		thisName := names[i]
		var previousName uint64 = 0
		if i > 0 {
			previousName = names[i-1]
		}
		var spacing uint64
		if spacingStrategy == "linear" {
			spacing = thisName - previousName
		} else if spacingStrategy == "xordistance" {
			spacing = thisName ^ previousName
		} else {
			panic("unknown spacing strategy")
		}
		if spacing > maxSpacing {
			maxSpacing = spacing
			minName = previousName
			maxName = thisName
		}
	}
	// check the space between the last node and MaxUint64
	lastName := names[len(names)-1]
	lastSpacing := math.MaxUint64 - lastName
	if lastSpacing > maxSpacing {
		minName = lastName
		maxName = math.MaxUint64
	}
	// find a new name within this spacing
	for name <= minName && name >= maxName {
		name = rand.Uint64()
	}
	return name
}

func nameForQuiestestHalf(names []uint64) uint64 {
	// count the vaults in each half
	var halfway uint64 = math.MaxUint64 / 2
	firstHalfVaults := 0
	secondHalfVaults := 0
	for _, name := range names {
		if name < halfway {
			firstHalfVaults = firstHalfVaults + 1
		} else {
			secondHalfVaults = secondHalfVaults + 1
		}
	}
	var minName uint64 = 0
	var maxName uint64 = math.MaxUint64
	if firstHalfVaults > secondHalfVaults {
		minName = halfway
	} else {
		maxName = halfway
	}
	// find a new name within this spacing
	name := rand.Uint64()
	for name <= minName && name >= maxName {
		name = rand.Uint64()
	}
	return name
}

func nameStr(i uint64) string {
	// base32 name truncated to first 7 characters
	u64b32chars := 13
	s := strconv.FormatUint(i, 32)
	for len(s) < u64b32chars {
		s = "0" + s
	}
	return s[0:7]
}

func standardDeviation(numbers []uint64) int64 {
	avg := average(numbers)
	bigAvg := big.NewInt(0).SetUint64(avg)
	totalDiffs := big.NewInt(0)
	for _, number := range numbers {
		bigNumber := big.NewInt(0).SetUint64(number)
		bigDiff := big.NewInt(0).Sub(bigNumber, bigAvg)
		bigDiffSquared := big.NewInt(0).Mul(bigDiff, bigDiff)
		totalDiffs = big.NewInt(0).Add(totalDiffs, bigDiffSquared)
	}
	bigDeviation := totalDiffs.Div(totalDiffs, big.NewInt(int64(len(numbers)-1)))
	return bigDeviation.Sqrt(bigDeviation).Int64()
}

func average(numbers []uint64) uint64 {
	total := big.NewInt(0)
	for _, number := range numbers {
		bigNumber := big.NewInt(0).SetUint64(number)
		total = total.Add(total, bigNumber)
	}
	bigLen := big.NewInt(int64(len(numbers)))
	bigAverage := total.Div(total, bigLen)
	return bigAverage.Uint64()
}
