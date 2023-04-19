package smpls

import (
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

// expVals contains the expected values for the Stat
type statTC struct {
	testhelper.ID
	values     []float64
	expMin     float64
	expMeanMin float64
	expMean    float64
	expSD      float64
	expMax     float64
	expMeanMax float64
}

func cmpWithExpected(t *testing.T, s *Stat, tc statTC) {
	t.Helper()

	id := tc.IDStr()
	min, meanMin, mean, sd, max, meanMax, count := s.Vals()
	testhelper.DiffFloat(t, id, "min", min, tc.expMin, 0.0)
	testhelper.DiffFloat(t, id, "mean min", meanMin, tc.expMeanMin, 0.0)
	testhelper.DiffFloat(t, id, "mean", mean, tc.expMean, 0.0)
	testhelper.DiffFloat(t, id, "sd", sd, tc.expSD, 0.00001)
	testhelper.DiffFloat(t, id, "max", max, tc.expMax, 0.0)
	testhelper.DiffFloat(t, id, "mean max", meanMax, tc.expMeanMax, 0.0)
	testhelper.DiffInt(t, id, "count", count, len(tc.values))

	id += " - comparing against the individual funcs"
	testhelper.DiffFloat(t, id, "min", min, s.Min(), 0.0)
	testhelper.DiffFloat(t, id, "mean min", meanMin, s.MeanMin(), 0.0)
	testhelper.DiffFloat(t, id, "mean", mean, s.Mean(), 0.0)
	testhelper.DiffFloat(t, id, "sd", sd, s.StdDev(), 0.00001)
	testhelper.DiffFloat(t, id, "max", max, s.Max(), 0.0)
	testhelper.DiffFloat(t, id, "mean max", meanMax, s.MeanMax(), 0.0)
	testhelper.DiffInt(t, id, "count", count, s.Count())
}

func TestStat(t *testing.T) {
	testCases := []statTC{
		{
			ID:         testhelper.MkID("3 values"),
			values:     []float64{1.0, 2.0, 3.0},
			expMin:     1.0,
			expMeanMin: 2.0,
			expMean:    2.0,
			expSD:      0.81649658,
			expMax:     3.0,
			expMeanMax: 2.0,
		},
		{
			ID: testhelper.MkID("22 values"),
			values: []float64{
				1.0,
				2.0, 2.0, 2.0, 2.0, 2.0,
				2.0, 2.0, 2.0, 2.0, 2.0,
				2.0, 2.0, 2.0, 2.0, 2.0,
				2.0, 2.0, 2.0, 2.0, 2.0,
				3.0,
			},
			expMin:     1.0,
			expMeanMin: 1.95,
			expMean:    2.0,
			expSD:      0.3015113,
			expMax:     3.0,
			expMeanMax: 2.05,
		},
	}

	for _, tc := range testCases {
		s, err := NewStat("unit")
		if err != nil {
			t.Fatal("Couldn't create the Stat value:", err)
		}

		for _, val := range tc.values {
			s.Add(val)
		}
		cmpWithExpected(t, s, tc)

		s.Reset()
		s.AddVals(tc.values...)
		cmpWithExpected(t, s, tc)
	}
}

// expectedCacheEntries returns the expected number of entries in the cache
func expectedCacheEntries(size, count int) int {
	cacheSize := size
	if cacheSize == 0 {
		cacheSize = dfltCacheSize
	}
	if count == 0 || count > cacheSize {
		return cacheSize
	}
	return count
}

// populateTestCache adds entries to the Stat's cache.
func populateTestCache(s *Stat, init, incr float64, count int) {
	v := init
	if count <= 0 || count > cap(s.cache) {
		count = cap(s.cache)
	}
	for i := 0; i < count; i++ {
		s.Add(v)
		v += incr
	}
}

func TestHist(t *testing.T) {
	testCases := []struct {
		testhelper.ID

		cacheSize  int
		cacheInit  float64
		cacheIncr  float64
		cacheCount int

		init  float64
		incr  float64
		count int

		expUnderflow       int
		expOverflow        int
		exp1stBucketCount  int
		expLastBucketCount int

		expBucketStart float64
		expBucketWidth float64
	}{
		{
			ID:                 testhelper.MkID("only cache values"),
			cacheInit:          180.0,
			cacheIncr:          6.0,
			expUnderflow:       0,
			expOverflow:        0,
			exp1stBucketCount:  dfltCacheSize / dfltHistBucketCount,
			expLastBucketCount: dfltCacheSize / dfltHistBucketCount,
			expBucketStart:     180.0,
			expBucketWidth: histBucketWidthScale *
				(6.0 * (dfltCacheSize - 1)) / dfltHistBucketCount,
		},
		{
			ID:                 testhelper.MkID("half cache"),
			cacheSize:          1000,
			cacheInit:          180.0,
			cacheIncr:          6.0,
			cacheCount:         500,
			expUnderflow:       0,
			expOverflow:        0,
			exp1stBucketCount:  500 / dfltHistBucketCount,
			expLastBucketCount: 500 / dfltHistBucketCount,
			expBucketStart:     180.0,
			expBucketWidth: histBucketWidthScale *
				(6.0 * (500 - 1)) / (dfltHistBucketCount),
		},
		{
			ID:                 testhelper.MkID("3 values below bucketStart"),
			cacheInit:          180.0,
			cacheIncr:          6.0,
			init:               60.0,
			incr:               20.0,
			count:              3,
			expUnderflow:       3,
			expOverflow:        0,
			exp1stBucketCount:  dfltCacheSize / dfltHistBucketCount,
			expLastBucketCount: dfltCacheSize / dfltHistBucketCount,
			expBucketStart:     180.0,
			expBucketWidth: histBucketWidthScale *
				(6.0 * (dfltCacheSize - 1)) / dfltHistBucketCount,
		},
		{
			ID:                 testhelper.MkID("3 values above end of hist"),
			cacheInit:          180.0,
			cacheIncr:          6.0,
			init:               6.0*dfltCacheSize + 180.0,
			incr:               20.0,
			count:              3,
			expUnderflow:       0,
			expOverflow:        3,
			exp1stBucketCount:  dfltCacheSize / dfltHistBucketCount,
			expLastBucketCount: dfltCacheSize / dfltHistBucketCount,
			expBucketStart:     180.0,
			expBucketWidth: histBucketWidthScale *
				(6.0 * (dfltCacheSize - 1)) / dfltHistBucketCount,
		},
	}

	for _, tc := range testCases {
		var s *Stat
		var err error
		SOFuncs := []StatOpt{}
		if tc.cacheSize > 0 {
			SOFuncs = append(SOFuncs, StatCacheSize(tc.cacheSize))
		}
		s, err = NewStat("units", SOFuncs...)
		if err != nil {
			t.Fatal("couldn't create the Stat:", err)
		}

		populateTestCache(s, tc.cacheInit, tc.cacheIncr, tc.cacheCount)
		v := tc.init
		for i := 0; i < tc.count; i++ {
			s.Add(v)
			v += tc.incr
		}

		s.populateHist()

		testhelper.DiffInt(t, tc.IDStr(), "count",
			s.count, expectedCacheEntries(tc.cacheSize, tc.cacheCount)+tc.count)
		testhelper.DiffInt(t, tc.IDStr(), "underflow",
			s.underflow, tc.expUnderflow)
		testhelper.DiffInt(t, tc.IDStr(), "overflow",
			s.overflow, tc.expOverflow)
		testhelper.DiffInt(t, tc.IDStr(), "1stBucketCount",
			s.hist[0], tc.exp1stBucketCount)
		testhelper.DiffInt(t, tc.IDStr(), "LastBucketCount",
			s.hist[len(s.hist)-1], tc.expLastBucketCount)
		testhelper.DiffFloat(t, tc.IDStr(), "bucket start",
			s.bucketStart, tc.expBucketStart, 0.0)
		testhelper.DiffFloat(t, tc.IDStr(), "bucket width",
			s.bucketWidth, tc.expBucketWidth, 0.00001)
	}
}

func TestInsert(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		v       float64
		vals    []float64
		discard discardType
		expVals []float64
	}{
		{
			ID:      testhelper.MkID("less than the biggest - min vals"),
			v:       1.23,
			vals:    []float64{1.1, 1.2, 1.3},
			discard: dropFromEnd,
			expVals: []float64{1.1, 1.2, 1.23},
		},
		{
			ID:      testhelper.MkID("less than the smallest - min vals"),
			v:       1.0,
			vals:    []float64{1.1, 1.2, 1.3},
			discard: dropFromEnd,
			expVals: []float64{1.0, 1.1, 1.2},
		},
		{
			ID:      testhelper.MkID("less than the 2nd smallest - min vals"),
			v:       1.11,
			vals:    []float64{1.1, 1.2, 1.3},
			discard: dropFromEnd,
			expVals: []float64{1.1, 1.11, 1.2},
		},
		{
			ID:      testhelper.MkID("bigger than the biggest - max vals"),
			v:       1.4,
			vals:    []float64{1.1, 1.2, 1.3},
			discard: dropFromStart,
			expVals: []float64{1.2, 1.3, 1.4},
		},
		{
			ID:      testhelper.MkID("bigger than the smallest - max vals"),
			v:       1.11,
			vals:    []float64{1.1, 1.2, 1.3},
			discard: dropFromStart,
			expVals: []float64{1.11, 1.2, 1.3},
		},
		{
			ID:      testhelper.MkID("bigger than the 2nd biggest - max vals"),
			v:       1.21,
			vals:    []float64{1.1, 1.2, 1.3},
			discard: dropFromStart,
			expVals: []float64{1.2, 1.21, 1.3},
		},
	}

	for _, tc := range testCases {
		initVals := make([]float64, len(tc.vals))
		copy(initVals, tc.vals)
		insert(tc.v, tc.vals, tc.discard)
		if floatSliceDiffers(tc.vals, tc.expVals) {
			t.Log(tc.IDStr())
			t.Logf("\t: inserting %g into %v\n", tc.v, initVals)
			if tc.discard == dropFromEnd {
				t.Log("\t\tDiscarding from the end\n")
			} else {
				t.Log("\t\tDiscarding from the start\n")
			}
			t.Log("\t: expected:", tc.expVals)
			t.Log("\t:      got:", tc.vals)
			t.Errorf("\t: unexpected result\n")
		}
	}
}

// floatSliceDiffers returns true if the two slices differ
func floatSliceDiffers(a, b []float64) bool {
	if len(a) != len(b) {
		return true
	}
	for i, aval := range a {
		if aval != b[i] {
			return true
		}
	}
	return false
}
