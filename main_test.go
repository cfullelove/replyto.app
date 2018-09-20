package main

import (
	"testing"
	"time"
)

type addressToReplyTimeCase struct {
	address  string
	current  time.Time
	expected time.Time
}

func (tc addressToReplyTimeCase) Equals(actual time.Time) bool {
	return tc.expected.Sub(actual) < time.Minute
}

func TestAddresses(t *testing.T) {
	addresses := []addressToReplyTimeCase{
		addressToReplyTimeCase{
			address:  "5d@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 6, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "1day@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 2, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "3days@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 4, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "3hrs@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 1, 3, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "3h@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 1, 3, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "3hours@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 1, 3, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "monday@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 8, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "monday@address.com",
			current:  time.Date(2018, 9, 21, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 9, 24, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "tuesday@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 2, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "wednesday@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 3, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "thursday@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 4, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "friday@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 5, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "saturday@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 6, 0, 0, 0, 0, time.Local),
		},
		addressToReplyTimeCase{
			address:  "sunday@address.com",
			current:  time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local),
			expected: time.Date(2018, 1, 7, 0, 0, 0, 0, time.Local),
		},
	}

	t.Parallel()

	for _, c := range addresses {
		t.Run(c.address, func(t *testing.T) {
			t.Logf("Beginning %v, expecting %v", c.address, c.expected.Format(time.ANSIC))
			rt, err := GetReplytime(c.address, c.current)
			if err != nil {
				t.Errorf("got and error and shouldn't have: %v", err)
				return
			}

			if c.expected != rt {
				t.Errorf("times don't match. wanted %v and %v", c.expected.Format(time.ANSIC), rt.Format(time.ANSIC))
				return
			}
		})
	}
}
