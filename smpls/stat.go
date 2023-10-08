package smpls

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/nickwells/mathutil.mod/v2/mathutil"
)

// Created: Thu Aug  6 13:01:57 2020

const (
	dfltMinMaxCount = 20
	minMinMaxCount  = 1

	dfltHistBucketCount  = 50
	minHistBucketCount   = 2
	histBucketWidthScale = 1.000001

	dfltCacheSize = 10000
	minCacheSize  = 2
)

type discardType int

const (
	dropFromStart discardType = iota
	dropFromEnd
)

// Stat records statistics. It will automatically calculate minima, maxima,
// mean and standard deviation. It will construct a histogram giving an
// indication of the distribution of values.
//
// Note that operations on this are not thread safe and it should be mutex
// protected if it is going to be updated by multiple threads.
type Stat struct {
	units string

	sum   float64
	sumSq float64
	count int
	mins  []float64
	maxs  []float64

	cache []float64

	underflow   int
	hist        []int
	overflow    int
	bucketStart float64
	bucketWidth float64

	histSizeChosen bool
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

// Count returns the number of values that have been added
func (s Stat) Count() int {
	return s.count
}

// Sum returns the sum of values that have been added
func (s Stat) Sum() float64 {
	return s.sum
}

// Min returns the min of the collected values or 0.0 if no values have
// been added
func (s Stat) Min() float64 {
	if s.count == 0 {
		return 0.0
	}
	return s.mins[0]
}

// MeanMin returns the mean of the N smallest collected values or 0.0 if no
// values have been added.
func (s Stat) MeanMin() float64 {
	if s.count == 0 {
		return 0.0
	}
	return calcMean(s.mins)
}

// Max returns the max of the collected values or 0.0 if no values have
// been added
func (s Stat) Max() float64 {
	if s.count == 0 {
		return 0.0
	}
	return s.maxs[len(s.maxs)-1]
}

// MeanMax returns the mean of the N largest collected values or 0.0 if no
// values have been added.
func (s Stat) MeanMax() float64 {
	if s.count == 0 {
		return 0.0
	}
	return calcMean(s.maxs)
}

// Mean returns the mean of the collected values or 0.0 if no values have
// been added
func (s Stat) Mean() float64 {
	if s.count == 0 {
		return 0.0
	}
	return s.sum / float64(s.count)
}

// StdDev returns the standard deviation of the collected values or 0.0 if
// fewer than 2 values have been added
func (s Stat) StdDev() float64 {
	if s.count < 2 {
		return 0.0
	}

	avg := s.sum / float64(s.count)
	return math.Sqrt((s.sumSq / float64(s.count)) - (avg * avg))
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
	if s.count < cap(s.cache) {
		s.populateHist()
	}

	if s.count < len(s.hist) {
		return ""
	}

	countFmt := fmt.Sprintf("%%%dd", mathutil.Digits(int64(s.count))) +
		" %6.2f%% %s"

	width, precision := mathutil.FmtValsForSigFigsMulti(3,
		s.bucketStart,
		s.bucketWidth,
		s.bucketStart+s.bucketWidth*float64(len(s.hist)))
	valFmt := fmt.Sprintf("%%%d.%df", width, precision)
	valSpace := strings.Repeat(" ", width)
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

// StatCacheSize returns a function that will create the cache slice of the
// given size in a Stat object
func StatCacheSize(c int) StatOpt {
	return func(s *Stat) error {
		if s.cache != nil {
			return errors.New(
				"the cache of values has already been created")
		}
		if c < minCacheSize {
			return fmt.Errorf(
				"Invalid cache size (%d) - it must be >= %d",
				c, minCacheSize)
		}

		s.cache = make([]float64, 0, c)
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
		s.histSizeChosen = true

		return nil
	}
}

// makeDfltHist creates a hist slice of default size if not already
// created. Note that it makes it with length set so that the slice is
// populated with zero initial values.
func (s *Stat) makeDfltHist() {
	if s.hist == nil {
		s.hist = make([]int, dfltHistBucketCount)
	}
}

// makeDfltCache creates a cache slice of default size if not already created
func (s *Stat) makeDfltCache() {
	if s.cache == nil {
		s.cache = make([]float64, 0, dfltCacheSize)
	}
}

// makeDfltMinsMaxs creates the mins and maxs slices of default size if not
// already created
func (s *Stat) makeDfltMinsMaxs() {
	if s.mins == nil {
		s.mins = make([]float64, 0, dfltMinMaxCount)
		s.maxs = make([]float64, 0, dfltMinMaxCount)
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

	s.makeDfltCache()
	s.makeDfltMinsMaxs()
	s.makeDfltHist()

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

// resetIntSlice resets the contents of the slice to zeros
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

	resetFloat64Slice(s.cache)

	s.underflow = 0
	resetIntSlice(s.hist)
	s.overflow = 0
	s.bucketStart = 0
	s.bucketWidth = 0
}

// Add adds at least one new value to the Stat
func (s *Stat) Add(v float64, vals ...float64) {
	s.addVal(v)
	for _, v := range vals {
		s.addVal(v)
	}
}

// AddVals adds new values to the Stat
//
// Deprecated: Use Add, you can add multiple values
func (s *Stat) AddVals(vals ...float64) {
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

	if len(s.cache) < cap(s.cache) {
		s.cache = append(s.cache, v)

		if len(s.cache) == cap(s.cache) {
			s.populateHist()
		}
	} else {
		s.addToHist(v)
	}
}

// populateHist calculates the boundaries of the histogram and the bucket
// size and then populates the buckets from the cache
func (s *Stat) populateHist() {
	if s.cache == nil {
		return
	}

	s.makeDfltHist()

	s.initHist()

	for _, v := range s.cache {
		s.addToHist(v)
	}
	s.cache = nil
}

// initHist initialises the histogram. Unless the hist size has been chosen
// it will resize the histogram to ensure there are enough data values to
// have at least a minimum average number of entries in each bucket.  It sets
// the bucket start and bucket width values for the histogram.
func (s *Stat) initHist() {
	const minPerBucket = 5

	if !s.histSizeChosen {
		if s.count/len(s.hist) < minPerBucket {
			newHistSize := int(s.count / minPerBucket)
			if newHistSize < minHistBucketCount {
				newHistSize = minHistBucketCount
			}
			s.hist = s.hist[:newHistSize]
		}
	}

	s.bucketStart = s.mins[0]
	valRange := s.maxs[len(s.maxs)-1] - s.bucketStart
	bucketCount := float64(len(s.hist))
	s.bucketWidth = histBucketWidthScale * valRange / bucketCount
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

	switch discard {
	case dropFromEnd:
		for i, cmp = range vals {
			if cmp >= v {
				break
			}
		}

		if i+1 < len(vals) {
			copy(vals[i+1:], vals[i:len(vals)-1])
		}
	case dropFromStart:
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
