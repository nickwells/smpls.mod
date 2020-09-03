package smpls

import (
	"testing"

	"github.com/nickwells/testhelper.mod/testhelper"
)

func TestStat(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		values     []float64
		expMin     float64
		expMeanMin float64
		expAvg     float64
		expSD      float64
		expMax     float64
		expMeanMax float64
	}{
		{
			ID:         testhelper.MkID("3 values"),
			values:     []float64{1.0, 2.0, 3.0},
			expMin:     1.0,
			expMeanMin: 2.0,
			expAvg:     2.0,
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
			expAvg:     2.0,
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
		min, meanMin, avg, sd, max, meanMax, count := s.Vals()
		testhelper.DiffFloat64(t, tc.IDStr(), "min", min, tc.expMin, 0.0)
		testhelper.DiffFloat64(t, tc.IDStr(), "mean min",
			meanMin, tc.expMeanMin, 0.0)
		testhelper.DiffFloat64(t, tc.IDStr(), "avg", avg, tc.expAvg, 0.0)
		testhelper.DiffFloat64(t, tc.IDStr(), "sd", sd, tc.expSD, 0.00001)
		testhelper.DiffFloat64(t, tc.IDStr(), "max", max, tc.expMax, 0.0)
		testhelper.DiffFloat64(t, tc.IDStr(), "mean max",
			meanMax, tc.expMeanMax, 0.0)
		testhelper.DiffInt(t, tc.IDStr(), "count", count, len(tc.values))
	}
}

func TestHist(t *testing.T) {
	testCases := []struct {
		testhelper.ID

		cacheInit float64
		cacheIncr float64
		init      float64
		incr      float64
		count     int

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
			exp1stBucketCount:  cacheSize / dfltHistBucketCount,
			expLastBucketCount: cacheSize / dfltHistBucketCount,
			expBucketStart:     180.0,
			expBucketWidth: histBucketWidthScale *
				(6.0 * (cacheSize - 1)) / dfltHistBucketCount,
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
			exp1stBucketCount:  cacheSize / dfltHistBucketCount,
			expLastBucketCount: cacheSize / dfltHistBucketCount,
			expBucketStart:     180.0,
			expBucketWidth: histBucketWidthScale *
				(6.0 * (cacheSize - 1)) / dfltHistBucketCount,
		},
		{
			ID:                 testhelper.MkID("3 values above end of hist"),
			cacheInit:          180.0,
			cacheIncr:          6.0,
			init:               6.0*cacheSize + 180.0,
			incr:               20.0,
			count:              3,
			expUnderflow:       0,
			expOverflow:        3,
			exp1stBucketCount:  cacheSize / dfltHistBucketCount,
			expLastBucketCount: cacheSize / dfltHistBucketCount,
			expBucketStart:     180.0,
			expBucketWidth: histBucketWidthScale *
				(6.0 * (cacheSize - 1)) / dfltHistBucketCount,
		},
	}

	for _, tc := range testCases {
		s, err := NewStat("units")
		if err != nil {
			t.Fatal("couldn't create the Stat:", err)
		}
		v := tc.cacheInit
		for i := 0; i < len(s.cache); i++ {
			s.Add(v)
			v += tc.cacheIncr
		}
		v = tc.init
		for i := 0; i < tc.count; i++ {
			s.Add(v)
			v += tc.incr
		}

		testhelper.DiffInt(t, tc.IDStr(), "count",
			s.count, len(s.cache)+tc.count)
		testhelper.DiffInt(t, tc.IDStr(), "underflow",
			s.underflow, tc.expUnderflow)
		testhelper.DiffInt(t, tc.IDStr(), "overflow",
			s.overflow, tc.expOverflow)
		testhelper.DiffInt(t, tc.IDStr(), "1stBucketCount",
			s.hist[0], tc.exp1stBucketCount)
		testhelper.DiffInt(t, tc.IDStr(), "LastBucketCount",
			s.hist[len(s.hist)-1], tc.expLastBucketCount)
		testhelper.DiffFloat64(t, tc.IDStr(), "bucket start",
			s.bucketStart, tc.expBucketStart, 0.0)
		testhelper.DiffFloat64(t, tc.IDStr(), "bucket width",
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
