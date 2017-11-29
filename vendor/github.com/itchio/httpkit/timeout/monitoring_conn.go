package timeout

import (
	"net"
	"time"
)

var lastBandwidthUpdate time.Time
var bytesSinceLastUpdate float64
var maxBucketDuration = 1 * time.Second
var bps float64

type BytesReadFunc func(bytesRead int64)

var bytesReadCallback BytesReadFunc

type monitoringConn struct {
	Conn net.Conn
}

var _ net.Conn = (*monitoringConn)(nil)

func (mc *monitoringConn) Close() error {
	return mc.Conn.Close()
}

func (mc *monitoringConn) LocalAddr() net.Addr {
	return mc.Conn.LocalAddr()
}

func (mc *monitoringConn) RemoteAddr() net.Addr {
	return mc.Conn.RemoteAddr()
}

func (mc *monitoringConn) SetDeadline(t time.Time) error {
	return mc.Conn.SetDeadline(t)
}

func (mc *monitoringConn) SetReadDeadline(t time.Time) error {
	return mc.Conn.SetReadDeadline(t)
}

func (mc *monitoringConn) SetWriteDeadline(t time.Time) error {
	return mc.Conn.SetWriteDeadline(t)
}

func (mc *monitoringConn) Read(buf []byte) (int, error) {
	readBytes, err := mc.Conn.Read(buf)
	recordBytesRead(int64(readBytes))
	return readBytes, err
}

func (mc *monitoringConn) Write(buf []byte) (int, error) {
	return mc.Conn.Write(buf)
}

func recordBytesRead(bytesRead int64) {
	if bytesRead == 0 {
		return
	}

	bytesSinceLastUpdate += float64(bytesRead)
	if lastBandwidthUpdate.IsZero() {
		lastBandwidthUpdate = time.Now()
	}

	bucketDuration := time.Since(lastBandwidthUpdate)

	if bucketDuration > maxBucketDuration {
		bps = bytesSinceLastUpdate / bucketDuration.Seconds()
		lastBandwidthUpdate = time.Now()
		bytesSinceLastUpdate = 0.0
	}
}

func GetBPS() float64 {
	return bps
}
