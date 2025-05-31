package limiter

import (
	"net"
	"net/http"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		request    *http.Request
		expectedIP string
	}{
		{
			name: "X-Real-IP only",
			request: &http.Request{
				RemoteAddr: "127.0.0.1:12345", // Use localhost to make it trusted
				Header: http.Header{
					"X-Real-Ip": []string{"2.2.2.2"},
				},
			},
			expectedIP: "2.2.2.2",
		},
		{
			name: "X-Forwarded-For only",
			request: &http.Request{
				RemoteAddr: "192.168.1.1:12345", // Use private IP to make it trusted
				Header: http.Header{
					"X-Forwarded-For": []string{"3.3.3.3, 4.4.4.4"},
				},
			},
			expectedIP: "4.4.4.4",
		},
		{
			name: "RemoteAddr fallback",
			request: &http.Request{
				RemoteAddr: "1.1.1.1:12345",
			},
			expectedIP: "1.1.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := getClientIP(tt.request)
			if ip != tt.expectedIP {
				t.Errorf("getClientIP() = %v, want %v", ip, tt.expectedIP)
			}
		})
	}
}

func TestIPMatchesList(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		cidrs    []string
		expected bool
	}{
		{
			name:     "IP matches exact entry",
			ip:       "192.168.1.1",
			cidrs:    []string{"192.168.1.1"},
			expected: true,
		},
		{
			name:     "IP matches CIDR",
			ip:       "192.168.1.1",
			cidrs:    []string{"192.168.1.0/24"},
			expected: true,
		},
		{
			name:     "IP doesn't match",
			ip:       "192.168.1.1",
			cidrs:    []string{"10.0.0.0/8"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := ipMatchesList(ip, tt.cidrs)
			if result != tt.expected {
				t.Errorf("ipMatchesList(%v, %v) = %v, want %v", tt.ip, tt.cidrs, result, tt.expected)
			}
		})
	}
}

func TestGetNetwork(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		ipv4Prefix int
		ipv6Prefix int
		expected   string
	}{
		{
			name:       "IPv4 with /24",
			ip:         "192.168.1.1",
			ipv4Prefix: 24,
			ipv6Prefix: 64,
			expected:   "192.168.1.0/24",
		},
		{
			name:       "IPv6 with /64",
			ip:         "2001:db8::1",
			ipv4Prefix: 24,
			ipv6Prefix: 64,
			expected:   "2001:db8::/64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			network := getNetwork(ip, tt.ipv4Prefix, tt.ipv6Prefix)
			if network.String() != tt.expected {
				t.Errorf("getNetwork(%v, %v, %v) = %v, want %v",
					tt.ip, tt.ipv4Prefix, tt.ipv6Prefix, network.String(), tt.expected)
			}
		})
	}
}
