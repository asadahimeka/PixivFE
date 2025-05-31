package limiter

/*
TODO: these tests are broken at the moment due to the routes.BlockPage calls,
			which cause a panic since we haven't initialized Jet
*/

// import (
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// )

// func TestLimiter(t *testing.T) {
// 	tests := []struct {
// 		name           string
// 		path           string
// 		ip             string
// 		mockPassList   []string
// 		mockBlockList  []string
// 		expectedStatus int
// 		shouldCallNext bool
// 	}{
// 		{
// 			name:           "Static path should bypass all checks",
// 			path:           "/css/tailwind-style.css",
// 			ip:             "1.1.1.1",
// 			expectedStatus: http.StatusOK,
// 			shouldCallNext: true,
// 		},
// 		{
// 			name:           "Invalid IP should be rejected",
// 			path:           "/artworks/20",
// 			ip:             "invalid",
// 			expectedStatus: http.StatusTooManyRequests,
// 			shouldCallNext: false,
// 		},
// 		{
// 			name:           "Passed IP should bypass checks",
// 			path:           "/artworks/20",
// 			ip:             "1.1.1.1",
// 			mockPassList:   []string{"1.1.1.1/32"},
// 			expectedStatus: http.StatusOK,
// 			shouldCallNext: true,
// 		},
// 		{
// 			name:           "Blocked IP should be rejected",
// 			path:           "/artworks/20",
// 			ip:             "1.1.1.1",
// 			mockBlockList:  []string{"1.1.1.1/32"},
// 			expectedStatus: http.StatusTooManyRequests,
// 			shouldCallNext: false,
// 		},
// 		{
// 			name:           "Rate limit excluded path should bypass rate limiting",
// 			path:           "/about",
// 			ip:             "1.1.1.1",
// 			expectedStatus: http.StatusOK,
// 			shouldCallNext: true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Setup mock next handler to verify it's called
// 			nextCalled := false
// 			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 				nextCalled = true
// 			})

// 			// Setup test request
// 			req := httptest.NewRequest("GET", tt.path, nil)
// 			req.Header.Set("X-Real-IP", tt.ip)

// 			// Setup response recorder
// 			rr := httptest.NewRecorder()

// 			// Configure mock lists
// 			limiterCfg.PassIPs = tt.mockPassList
// 			limiterCfg.BlockIPs = tt.mockBlockList

// 			// Execute middleware
// 			handler := Limiter(next)
// 			handler.ServeHTTP(rr, req)

// 			// Verify response
// 			if status := rr.Code; status != tt.expectedStatus {
// 				t.Errorf("handler returned wrong status code: got %v want %v",
// 					status, tt.expectedStatus)
// 			}

// 			// Verify if next handler was called as expected
// 			if nextCalled != tt.shouldCallNext {
// 				t.Errorf("next handler called = %v, want %v", nextCalled, tt.shouldCallNext)
// 			}
// 		})
// 	}
// }
