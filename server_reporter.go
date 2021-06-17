// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

package grpc_prometheus

import (
	"log"
	"regexp"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serverReporter struct {
	metrics         *ServerMetrics
	rpcType         grpcType
	serviceName     string
	methodName      string
	startTime       time.Time
	errMsgMaxLength uint8
}

var (
	illegalMetricValueChars *regexp.Regexp
)

func init() {
	re, err := regexp.Compile(`[^\w]`)
	if err != nil {
		log.Fatal(err)
	}
	illegalMetricValueChars = re
}

func newServerReporter(m *ServerMetrics, rpcType grpcType, fullMethod string, errMsgMaxLength uint8) *serverReporter {
	r := &serverReporter{
		metrics:         m,
		rpcType:         rpcType,
		errMsgMaxLength: errMsgMaxLength,
	}
	if r.metrics.serverHandledHistogramEnabled {
		r.startTime = time.Now()
	}
	r.serviceName, r.methodName = splitMethodName(fullMethod)
	r.metrics.serverStartedCounter.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
	return r
}

func (r *serverReporter) ReceivedMessage() {
	r.metrics.serverStreamMsgReceived.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
}

func (r *serverReporter) SentMessage() {
	r.metrics.serverStreamMsgSent.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Inc()
}

func (r *serverReporter) Handled(code codes.Code) {
	r.metrics.serverHandledCounter.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName, code.String(), "").Inc()
	if r.metrics.serverHandledHistogramEnabled {
		r.metrics.serverHandledHistogram.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Observe(time.Since(r.startTime).Seconds())
	}
}

func (r *serverReporter) HandledWithStatus(st *status.Status) {
	r.metrics.serverHandledCounter.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName, st.Code().String(), r.slugifyError(st.Err())).Inc()
	if r.metrics.serverHandledHistogramEnabled {
		r.metrics.serverHandledHistogram.WithLabelValues(string(r.rpcType), r.serviceName, r.methodName).Observe(time.Since(r.startTime).Seconds())
	}
}

// slugifyError returns a string from an error, suitable for a prometheus metric label value
// we can follow some good practices for error messages
// - don't prefix error message with high cardinality/unique words
// - first 2-4 words should allow us to categorize the errors
func (r *serverReporter) slugifyError(err error) string {
	if (nil != err) && ("" != err.Error()) {
		return firstN(strings.ToLower(illegalMetricValueChars.ReplaceAllString(err.Error(), `_`)), r.errMsgMaxLength)
	} else {
		return ""
	}
}

// firstN return the first N chars of a string [unicode safe]
// https://stackoverflow.com/a/41604514/1308685
func firstN(s string, n uint8) string {
	i := uint8(0)
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}
