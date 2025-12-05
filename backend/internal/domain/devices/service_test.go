package devices

import "testing"

func TestDeviceService(t *testing.T) {
    s := NewDeviceService()
    if s == nil {
        t.Fatal("expected non-nil service")
    }
}
