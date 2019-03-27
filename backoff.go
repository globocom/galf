/*
* Go OAuth2 Client
*
* MIT License
*
* Copyright (c) 2015 Globo.com
 */

package galf

import (
	"math"
	"time"
)

// BackoffStrategy is used to determine how long a retry request should wait until attempted
type BackoffStrategy func(retry int) time.Duration

// ConstantBackOff always returns 30 Millisecond
func ConstantBackOff(_ int) time.Duration {
	return 30 * time.Millisecond
}

// ExponentialBackoff returns ever increasing backoffs by a power of 2 in seconds
func ExponentialBackoff(i int) time.Duration {
	return time.Duration(math.Pow(2, float64(i))) * time.Second
}

// LinearBackoff returns increasing durations, each a second longer than the last
// n seconds where n is the retry number
func LinearBackoff(i int) time.Duration {
	return time.Duration(i) * time.Second
}
