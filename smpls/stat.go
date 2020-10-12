package smpls

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Created: Thu Aug  6 13:01:57 2020

const (
	dfltMinMaxCount = 20
	minMinMaxCount  = 1

	dfltHistBucketCount  = 50
	minHistBucketCount   = 2
	histBucketWidthScale = 1.000001

	cacheSize = 10000
)

type discardType int

const (
	dropFromStart discardType = iota
	dropFromEnd
)

// Stat records statistics
type Stat struct {
	units string

	sum   float64
	sumSq float64
	count int
	mins  []float64
	maxs  []float64

	cache [cacheSize]float64

	underflow   int
	hist        []int
	overflow    int
	bucketStart float64
	bucketWidth float64
}

// calcMean will calculate the average value of the entries in the slice
// which must not be empty
func calcMean(s []float64) float64 {
	var sum float64
	for _, v := range s {
		sum += v
	}
	return sum / float64(len(s))
}

// Vals returns the calculated values from the stat.
//
// The meanMin and meanMax values are the means of the smallest and largest N
// values where N is the size of the caches of such values. These values
// should give a more stable indicator of the likely range of value than will
// be seen from the extreme values themselves. They will be identical to each
// other if the number of values added is less than N and will be identical
// to their non-smoothed counterparts if all the values are the same as each
// other. The cache size can be changed when creating the Stat object by
// passing the option function returned by StatMinMaxCount to the NewStat
// function.
func (s Stat) Vals() (min, meanMin, avg, sd, max, meanMax float64, count int) {
	if s.count == 0 {
		return
	}
	min = s.mins[0]
	meanMin = calcMean(s.mins)
	avg = s.sum / float64(s.count)
	sd = 0
	if s.count > 1 {
		sd = math.Sqrt((s.sumSq / float64(s.count)) - (avg * avg))
	}
	max = s.maxs[len(s.maxs)-1]
	meanMax = calcMean(s.maxs)
	count = s.count
	return
}

// String prints the statistics from the given values
func (s Stat) String() string {
	min, meanMin, avg, sd, max, meanMax, count := s.Vals()
	return fmt.Sprintf(
		"%7d observations,"+
			" min: %8.2e (%8.2e),"+
			" avg: %8.2e,"+
			" max: %8.2e (%8.2e),"+
			" SD: %8.2e",
		count, min, meanMin, avg, max, meanMax, sd)
}

// Hist returns a string showing the histogram of values
func (s Stat) Hist() string {
	if s.count < len(s.hist) {
		return ""
	}

	countFmt := fmt.Sprintf("%%%dd", maxDigits(s.count)) + " %6.2f%% %s"

	valFmt := "%8.2e"
	valSpace := strings.Repeat(" ", len(fmt.Sprintf(valFmt, 0.0)))
	fromFmt := ">= " + valFmt
	toFmt := "< " + valFmt

	underflowFmt := valSpace + "      " + toFmt + ": %s\n"
	overflowFmt := fromFmt + "     " + valSpace + ": %s\n"
	stdFmt := fromFmt + " , " + toFmt + ": %s\n"

	hist := "units: " + s.units + "\n"
	hist += fmt.Sprintf(underflowFmt, s.bucketStart,
		histValStr(s.underflow, s.count, countFmt))

	minVal := s.bucketStart
	maxVal := minVal + s.bucketWidth
	for _, count := range s.hist {
		hist += fmt.Sprintf(stdFmt, minVal, maxVal,
			histValStr(count, s.count, countFmt))
		minVal = maxVal
		maxVal += s.bucketWidth
	}

	hist += fmt.Sprintf(overflowFmt, minVal,
		histValStr(s.overflow, s.count, countFmt))
	return hist
}

// histValStr returns a string holding the formatted value. The value is
// shown, followed by the value as a percentage of the total and a string of
// stars corresponding to the percentage value
func histValStr(val, tot int, fmtStr string) string {
	pct := 100.0 * float64(val) / float64(tot)
	return fmt.Sprintf(fmtStr, val, pct, strings.Repeat("*", int(pct*0.5)))
}

// maxDigits returns the space (the number of digits plus potentially a sign
// marker) needed to print the value
func maxDigits(v int) int {
	if v == 0 {
		return 1
	}

	d := 0
	if v < 0 {
		d++
		v *= -1
	}
	d += int(math.Ceil(math.Log10(float64(v) + 0.1)))

	return d
}

type StatOpt func(s *Stat) error

// StatMinMaxCount returns a function that will create min/max slices of the
// given size in a Stat object
func StatMinMaxCount(c int) StatOpt {
	return func(s *Stat) error {
		if s.mins != nil {
			return errors.New(
				"the slice of minimum values has already been created")
		}
		if c < minMinMaxCount {
			return fmt.Errorf(
				"Invalid Min/Max Count (%d) - it must be >= %d",
				c, minMinMaxCount)
		}

		s.mins = make([]float64, 0, c)
		s.maxs = make([]float64, 0, c)
		return nil
	}
}

// StatHistBucketCount returns a function that will create a hist slice with the
// given number of buckets in a Stat object
func StatHistBucketCount(c int) StatOpt {
	return func(s *Stat) error {
		if s.hist != nil {
			return errors.New(
				"the histogram slice has already been created")
		}
		if c < minHistBucketCount {
			return fmt.Errorf(
				"Invalid Hist Bucket Count (%d) - it must be >= %d",
				c, minHistBucketCount)
		}

		s.hist = make([]int, c)
		return nil
	}
}

// NewStat creates a new instance of a Stat
func NewStat(units string, opts ...StatOpt) (*Stat, error) {
	s := &Stat{units: units}

	for _, o := range opts {
		err := o(s)
		if err != nil {
			return nil, err
		}
	}
	if len(s.mins) == 0 {
		s.mins = make([]float64, 0, dfltMinMaxCount)
	}
	if len(s.maxs) == 0 {
		s.maxs = make([]float64, 0, dfltMinMaxCount)
	}
	if len(s.hist) == 0 {
		s.hist = make([]int, dfltHistBucketCount)
	}

	return s, nil
}

// NewStatOrPanic creates a new instance of a Stat and will panic if any
// errors are detected
func NewStatOrPanic(units string, opts ...StatOpt) *Stat {
	s, err := NewStat(units, opts...)
	if err != nil {
		panic(err)
	}
	return s
}

// resetFloat64Slice resets the contents of the slice to zeros
func resetFloat64Slice(s []float64) {
	if len(s) == 0 {
		return
	}
	zeros := make([]float64, len(s))
	copy(s, zeros)
}

// resetInySlice resets the contents of the slice to zeros
func resetIntSlice(s []int) {
	if len(s) == 0 {
		return
	}
	zeros := make([]int, len(s))
	copy(s, zeros)
}

// Reset resets the Stat back to its initial state
func (s *Stat) Reset() {
	s.sum = 0
	s.sumSq = 0
	s.count = 0
	s.mins = s.mins[:0]
	s.maxs = s.maxs[:0]

	resetFloat64Slice(s.cache[:])

	s.underflow = 0
	resetIntSlice(s.hist)
	s.overflow = 0
	s.bucketStart = 0
	s.bucketWidth = 0
}

// Add adds new values to the Stat
func (s *Stat) Add(v float64, vals ...float64) {
	s.addVal(v)
	for _, v := range vals {
		s.addVal(v)
	}
}

// addVal adds a single new value to the Stat
func (s *Stat) addVal(v float64) {
	maxIdx := cap(s.mins) - 1

	s.sum += v
	s.sumSq += v * v
	s.count++

	if s.count <= cap(s.mins) {
		s.mins = append(s.mins, v)
		s.maxs = append(s.maxs, v)
		sort.Float64s(s.mins)
		sort.Float64s(s.maxs)
	} else {
		if v < s.mins[maxIdx] { // smaller than the largest min value
			insert(v, s.mins, dropFromEnd)
		}
		if v > s.maxs[0] { // larger than the smallest max value
			insert(v, s.maxs, dropFromStart)
		}
	}

	if s.count <= len(s.cache) {
		s.cache[s.count-1] = v

		if s.count == len(s.cache) {
			s.populateHist()
		}
	} else {
		s.addToHist(v)
	}
}

// populateHist calculates the boundaries of the histogram and the bucket
// size and then populates the buckets from the cache
func (s *Stat) populateHist() {
	if len(s.hist) == 0 {
		panic("The histogram has not been initialised" +
			" - use NewStat(...) to create the Stat object")
	}

	s.bucketStart = s.mins[0]
	end := s.maxs[len(s.maxs)-1]
	s.bucketWidth =
		histBucketWidthScale *
			(end - s.bucketStart) / float64(len(s.hist))

	for _, v := range s.cache {
		s.addToHist(v)
	}
}

// addToHist adds the value to the histogram of values
func (s *Stat) addToHist(v float64) {
	idx := int(math.Floor((v - s.bucketStart) / s.bucketWidth))

	if idx < 0 {
		s.underflow++
		return
	}

	if idx >= len(s.hist) {
		s.overflow++
		return
	}

	s.hist[idx]++
}

// insert inserts the value into the slice of values shifting the remaining
// values along and discarding from one end or the other according to the
// discard type. The vals slice is assumed to be sorted in ascending order.
func insert(v float64, vals []float64, discard discardType) {
	var i int
	var cmp float64

	if discard == dropFromEnd {
		for i, cmp = range vals {
			if cmp >= v {
				break
			}
		}

		if i+1 < len(vals) {
			copy(vals[i+1:], vals[i:len(vals)-1])
		}
	} else if discard == dropFromStart {
		for i = len(vals) - 1; i > 0; i-- {
			if vals[i] < v {
				break
			}
		}
		if i > 0 {
			copy(vals[:i], vals[1:i+1])
		}
	}
	vals[i] = v
}
